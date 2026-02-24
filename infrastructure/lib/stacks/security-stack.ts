import * as cdk from 'aws-cdk-lib';
import * as kms from 'aws-cdk-lib/aws-kms';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import { Construct } from 'constructs';

export class SecurityStack extends cdk.Stack {
  public readonly kmsKey: kms.Key;
  public readonly stripeApiKeySecret: secretsmanager.Secret;
  public readonly redisAuthTokenSecret: secretsmanager.Secret;

  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // KMS Key for encryption at rest
    this.kmsKey = new kms.Key(this, 'TourAgentKmsKey', {
      description: 'KMS key for tour agent platform encryption',
      enableKeyRotation: true,
      alias: 'tour-agent-platform',
      removalPolicy: cdk.RemovalPolicy.RETAIN, // Retain key on stack deletion for security
    });

    // Stripe API key secret (placeholder - to be updated manually)
    this.stripeApiKeySecret = new secretsmanager.Secret(this, 'StripeApiKey', {
      secretName: 'tour-agent/stripe/api-key',
      description: 'Stripe API key for payment processing',
      secretStringValue: cdk.SecretValue.unsafePlainText('placeholder-update-after-deployment'),
      encryptionKey: this.kmsKey,
    });

    // Redis auth token secret
    this.redisAuthTokenSecret = new secretsmanager.Secret(this, 'RedisAuthToken', {
      secretName: 'tour-agent/redis/auth-token',
      description: 'ElastiCache Redis authentication token',
      generateSecretString: {
        excludePunctuation: true,
        includeSpace: false,
        passwordLength: 32,
      },
      encryptionKey: this.kmsKey,
    });

    // Outputs
    new cdk.CfnOutput(this, 'KmsKeyId', {
      value: this.kmsKey.keyId,
      description: 'KMS Key ID',
      exportName: 'TourAgentKmsKeyId',
    });

    new cdk.CfnOutput(this, 'KmsKeyArn', {
      value: this.kmsKey.keyArn,
      description: 'KMS Key ARN',
      exportName: 'TourAgentKmsKeyArn',
    });

    new cdk.CfnOutput(this, 'StripeApiKeySecretArn', {
      value: this.stripeApiKeySecret.secretArn,
      description: 'Stripe API key secret ARN',
      exportName: 'TourAgentStripeApiKeySecretArn',
    });

    new cdk.CfnOutput(this, 'RedisAuthTokenSecretArn', {
      value: this.redisAuthTokenSecret.secretArn,
      description: 'Redis auth token secret ARN',
      exportName: 'TourAgentRedisAuthTokenSecretArn',
    });
  }
}
