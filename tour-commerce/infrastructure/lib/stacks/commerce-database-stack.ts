import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as rds from 'aws-cdk-lib/aws-rds';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as cr from 'aws-cdk-lib/custom-resources';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as path from 'path';
import { Construct } from 'constructs';

export interface CommerceDatabaseStackProps extends cdk.StackProps {
  vpc: ec2.IVpc;
  lambdaSecurityGroup: ec2.ISecurityGroup;
  databaseSecurityGroup: ec2.ISecurityGroup;
}

export class CommerceDatabaseStack extends cdk.Stack {
  public readonly cluster: rds.DatabaseCluster;
  public readonly credentialsSecret: secretsmanager.Secret;

  constructor(scope: Construct, id: string, props: CommerceDatabaseStackProps) {
    super(scope, id, props);

    // ── credentials ───────────────────────────────────────────────────────────
    this.credentialsSecret = new secretsmanager.Secret(this, 'CommerceDbCredentials', {
      secretName: 'tour-commerce/database/credentials',
      description: 'Aurora Serverless v2 credentials for tour-commerce',
      generateSecretString: {
        secretStringTemplate: JSON.stringify({ username: 'commerce' }),
        generateStringKey: 'password',
        excludePunctuation: true,
        includeSpace: false,
        passwordLength: 32,
      },
    });

    // ── Aurora Serverless v2 cluster ──────────────────────────────────────────
    this.cluster = new rds.DatabaseCluster(this, 'CommerceCluster', {
      engine: rds.DatabaseClusterEngine.auroraPostgres({
        version: rds.AuroraPostgresEngineVersion.VER_15_4,
      }),
      credentials: rds.Credentials.fromSecret(this.credentialsSecret),
      defaultDatabaseName: 'tourcommerce',
      vpc: props.vpc,
      vpcSubnets: { subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS },
      securityGroups: [props.databaseSecurityGroup],
      serverlessV2MinCapacity: 0.5,
      serverlessV2MaxCapacity: 4,
      writer: rds.ClusterInstance.serverlessV2('writer'),
      readers: [
        rds.ClusterInstance.serverlessV2('reader', { scaleWithWriter: true }),
      ],
      backup: { retention: cdk.Duration.days(7) },
      deletionProtection: true,
      storageEncrypted: true,
    });

    // ── migration Lambda ───────────────────────────────────────────────────────
    const migratorFn = new lambda.Function(this, 'CommerceMigrator', {
      runtime: lambda.Runtime.NODEJS_20_X,
      handler: 'index.handler',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../database/schema-deployer-lambda')),
      vpc: props.vpc,
      vpcSubnets: { subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS },
      securityGroups: [props.lambdaSecurityGroup],
      environment: {
        DB_SECRET_ARN: this.credentialsSecret.secretArn,
        DB_ENDPOINT: this.cluster.clusterEndpoint.hostname,
        DB_NAME: 'tourcommerce',
      },
      timeout: cdk.Duration.minutes(5),
      memorySize: 256,
    });

    this.credentialsSecret.grantRead(migratorFn);
    this.cluster.connections.allowFrom(props.lambdaSecurityGroup, ec2.Port.tcp(5432));

    migratorFn.addToRolePolicy(new iam.PolicyStatement({
      actions: ['secretsmanager:GetSecretValue'],
      resources: [this.credentialsSecret.secretArn],
    }));

    // Custom resource triggers migration on deploy.
    new cr.AwsCustomResource(this, 'CommerceMigrationTrigger', {
      onCreate: {
        service: 'Lambda',
        action: 'invoke',
        parameters: {
          FunctionName: migratorFn.functionName,
          InvocationType: 'RequestResponse',
          Payload: JSON.stringify({ action: 'migrate' }),
        },
        physicalResourceId: cr.PhysicalResourceId.of('CommerceMigrationTrigger'),
      },
      policy: cr.AwsCustomResourcePolicy.fromStatements([
        new iam.PolicyStatement({
          actions: ['lambda:InvokeFunction'],
          resources: [migratorFn.functionArn],
        }),
      ]),
    });

    // ── outputs ───────────────────────────────────────────────────────────────
    new cdk.CfnOutput(this, 'CommerceDbEndpoint', {
      value: this.cluster.clusterEndpoint.hostname,
      exportName: 'TourCommerceDbEndpoint',
    });
    new cdk.CfnOutput(this, 'CommerceDbSecretArn', {
      value: this.credentialsSecret.secretArn,
      exportName: 'TourCommerceDbSecretArn',
    });
  }
}
