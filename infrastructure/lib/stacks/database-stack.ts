import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as rds from 'aws-cdk-lib/aws-rds';
import * as kms from 'aws-cdk-lib/aws-kms';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import { Construct } from 'constructs';

export interface DatabaseStackProps extends cdk.StackProps {
  vpc: ec2.Vpc;
  databaseSecurityGroup: ec2.SecurityGroup;
  kmsKey: kms.Key;
}

export class DatabaseStack extends cdk.Stack {
  public readonly cluster: rds.DatabaseCluster;
  public readonly databaseCredentialsSecret: secretsmanager.Secret;

  constructor(scope: Construct, id: string, props: DatabaseStackProps) {
    super(scope, id, props);

    // Database credentials secret
    this.databaseCredentialsSecret = new secretsmanager.Secret(this, 'DatabaseCredentials', {
      secretName: 'explorer-agent/database/credentials',
      description: 'Aurora Serverless v2 PostgreSQL credentials',
      generateSecretString: {
        secretStringTemplate: JSON.stringify({ username: 'postgres' }),
        generateStringKey: 'password',
        excludePunctuation: true,
        includeSpace: false,
        passwordLength: 32,
      },
      // Use AWS managed key to avoid circular dependency with Lambda functions
      // encryptionKey: props.kmsKey,
    });

    // Aurora Serverless v2 PostgreSQL cluster
    this.cluster = new rds.DatabaseCluster(this, 'ExplorerAgentDatabase', {
      engine: rds.DatabaseClusterEngine.auroraPostgres({
        version: rds.AuroraPostgresEngineVersion.VER_15_4,
      }),
      credentials: rds.Credentials.fromSecret(this.databaseCredentialsSecret),
      writer: rds.ClusterInstance.serverlessV2('writer', {
        autoMinorVersionUpgrade: true,
        publiclyAccessible: false,
      }),
      readers: [
        rds.ClusterInstance.serverlessV2('reader1', {
          scaleWithWriter: true,
          autoMinorVersionUpgrade: true,
          publiclyAccessible: false,
        }),
      ],
      vpc: props.vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
      securityGroups: [props.databaseSecurityGroup],
      storageEncrypted: true,
      storageEncryptionKey: props.kmsKey,
      defaultDatabaseName: 'exploreragent',
      backup: {
        retention: cdk.Duration.days(7),
        preferredWindow: '03:00-04:00', // UTC
      },
      preferredMaintenanceWindow: 'sun:04:00-sun:05:00', // UTC
      cloudwatchLogsExports: ['postgresql'],
      cloudwatchLogsRetention: 30, // days
      deletionProtection: true,
      removalPolicy: cdk.RemovalPolicy.SNAPSHOT,
      serverlessV2MinCapacity: 0.5, // ACU
      serverlessV2MaxCapacity: 128, // ACU
    });

    // Enable Performance Insights
    const cfnDbCluster = this.cluster.node.defaultChild as rds.CfnDBCluster;
    cfnDbCluster.enableCloudwatchLogsExports = ['postgresql'];

    // Outputs
    new cdk.CfnOutput(this, 'DatabaseClusterEndpoint', {
      value: this.cluster.clusterEndpoint.hostname,
      description: 'Aurora Serverless v2 cluster endpoint',
      exportName: 'ExplorerAgentDatabaseClusterEndpoint',
    });

    new cdk.CfnOutput(this, 'DatabaseClusterReadEndpoint', {
      value: this.cluster.clusterReadEndpoint.hostname,
      description: 'Aurora Serverless v2 read endpoint',
      exportName: 'ExplorerAgentDatabaseClusterReadEndpoint',
    });

    new cdk.CfnOutput(this, 'DatabaseName', {
      value: 'exploreragent',
      description: 'Database name',
      exportName: 'ExplorerAgentDatabaseName',
    });

    new cdk.CfnOutput(this, 'DatabasePort', {
      value: this.cluster.clusterEndpoint.port.toString(),
      description: 'Database port',
      exportName: 'ExplorerAgentDatabasePort',
    });

    new cdk.CfnOutput(this, 'DatabaseCredentialsSecretArn', {
      value: this.databaseCredentialsSecret.secretArn,
      description: 'Database credentials secret ARN',
      exportName: 'ExplorerAgentDatabaseCredentialsSecretArn',
    });
  }
}
