import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as kms from 'aws-cdk-lib/aws-kms';
import { Construct } from 'constructs';

export interface StorageStackProps extends cdk.StackProps {
  kmsKey: kms.Key;
}

export class StorageStack extends cdk.Stack {
  public readonly vouchersBucket: s3.Bucket;
  public readonly staticAssetsBucket: s3.Bucket;
  public readonly distribution: cloudfront.Distribution;

  constructor(scope: Construct, id: string, props: StorageStackProps) {
    super(scope, id, props);

    // S3 bucket for vouchers (PDFs with QR codes)
    this.vouchersBucket = new s3.Bucket(this, 'VouchersBucket', {
      bucketName: `tour-agent-vouchers-${this.account}`,
      encryption: s3.BucketEncryption.KMS,
      encryptionKey: props.kmsKey,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      versioned: true,
      lifecycleRules: [
        {
          id: 'DeleteOldVersions',
          noncurrentVersionExpiration: cdk.Duration.days(90),
        },
      ],
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      autoDeleteObjects: false,
    });

    // S3 bucket for static assets (images, documents)
    this.staticAssetsBucket = new s3.Bucket(this, 'StaticAssetsBucket', {
      bucketName: `tour-agent-assets-${this.account}`,
      encryption: s3.BucketEncryption.S3_MANAGED,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      versioned: false,
      lifecycleRules: [
        {
          id: 'TransitionToIA',
          transitions: [
            {
              storageClass: s3.StorageClass.INFREQUENT_ACCESS,
              transitionAfter: cdk.Duration.days(90),
            },
          ],
        },
      ],
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      autoDeleteObjects: false,
      cors: [
        {
          allowedMethods: [s3.HttpMethods.GET, s3.HttpMethods.HEAD],
          allowedOrigins: ['*'],
          allowedHeaders: ['*'],
          maxAge: 3600,
        },
      ],
    });

    // CloudFront Origin Access Identity for S3
    const originAccessIdentity = new cloudfront.OriginAccessIdentity(this, 'OAI', {
      comment: 'OAI for tour agent static assets',
    });

    // Grant CloudFront read access to static assets bucket
    this.staticAssetsBucket.grantRead(originAccessIdentity);

    // CloudFront distribution
    this.distribution = new cloudfront.Distribution(this, 'CDN', {
      comment: 'CDN for tour agent platform',
      defaultBehavior: {
        origin: origins.S3BucketOrigin.withOriginAccessIdentity(this.staticAssetsBucket, {
          originAccessIdentity,
        }),
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
        allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD_OPTIONS,
        cachedMethods: cloudfront.CachedMethods.CACHE_GET_HEAD_OPTIONS,
        compress: true,
        cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
      },
      priceClass: cloudfront.PriceClass.PRICE_CLASS_100, // US, Canada, Europe
      enableLogging: true,
      logBucket: new s3.Bucket(this, 'CloudFrontLogsBucket', {
        bucketName: `tour-agent-cloudfront-logs-${this.account}`,
        encryption: s3.BucketEncryption.S3_MANAGED,
        blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
        lifecycleRules: [
          {
            id: 'DeleteOldLogs',
            expiration: cdk.Duration.days(90),
          },
        ],
        removalPolicy: cdk.RemovalPolicy.DESTROY,
        autoDeleteObjects: true,
      }),
      logFilePrefix: 'cloudfront-logs/',
      minimumProtocolVersion: cloudfront.SecurityPolicyProtocol.TLS_V1_2_2021,
    });

    // Outputs
    new cdk.CfnOutput(this, 'VouchersBucketName', {
      value: this.vouchersBucket.bucketName,
      description: 'S3 bucket for vouchers',
      exportName: 'TourAgentVouchersBucketName',
    });

    new cdk.CfnOutput(this, 'VouchersBucketArn', {
      value: this.vouchersBucket.bucketArn,
      description: 'S3 bucket ARN for vouchers',
      exportName: 'TourAgentVouchersBucketArn',
    });

    new cdk.CfnOutput(this, 'StaticAssetsBucketName', {
      value: this.staticAssetsBucket.bucketName,
      description: 'S3 bucket for static assets',
      exportName: 'TourAgentStaticAssetsBucketName',
    });

    new cdk.CfnOutput(this, 'StaticAssetsBucketArn', {
      value: this.staticAssetsBucket.bucketArn,
      description: 'S3 bucket ARN for static assets',
      exportName: 'TourAgentStaticAssetsBucketArn',
    });

    new cdk.CfnOutput(this, 'CloudFrontDistributionId', {
      value: this.distribution.distributionId,
      description: 'CloudFront distribution ID',
      exportName: 'TourAgentCloudFrontDistributionId',
    });

    new cdk.CfnOutput(this, 'CloudFrontDomainName', {
      value: this.distribution.distributionDomainName,
      description: 'CloudFront distribution domain name',
      exportName: 'TourAgentCloudFrontDomainName',
    });
  }
}
