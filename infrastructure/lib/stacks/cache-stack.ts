import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as elasticache from 'aws-cdk-lib/aws-elasticache';
import * as kms from 'aws-cdk-lib/aws-kms';
import { Construct } from 'constructs';

export interface CacheStackProps extends cdk.StackProps {
  vpc: ec2.Vpc;
  cacheSecurityGroup: ec2.SecurityGroup;
  kmsKey: kms.Key;
}

export class CacheStack extends cdk.Stack {
  public readonly replicationGroup: elasticache.CfnReplicationGroup;

  constructor(scope: Construct, id: string, props: CacheStackProps) {
    super(scope, id, props);

    // Create subnet group for ElastiCache
    const subnetGroup = new elasticache.CfnSubnetGroup(this, 'RedisSubnetGroup', {
      description: 'Subnet group for ElastiCache Redis',
      subnetIds: props.vpc.selectSubnets({
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      }).subnetIds,
      cacheSubnetGroupName: 'explorer-agent-redis-subnet-group',
    });

    // ElastiCache Redis cluster with encryption
    this.replicationGroup = new elasticache.CfnReplicationGroup(this, 'RedisCluster', {
      replicationGroupId: 'explorer-agent-redis',
      replicationGroupDescription: 'Redis cluster for inventory locking',
      engine: 'redis',
      engineVersion: '7.0',
      cacheNodeType: 'cache.t4g.micro', // Start small, can scale up
      numCacheClusters: 2, // Primary + 1 replica for high availability
      automaticFailoverEnabled: true,
      multiAzEnabled: true,
      cacheSubnetGroupName: subnetGroup.cacheSubnetGroupName,
      securityGroupIds: [props.cacheSecurityGroup.securityGroupId],
      atRestEncryptionEnabled: true,
      kmsKeyId: props.kmsKey.keyId,
      transitEncryptionEnabled: true,
      authToken: cdk.Fn.sub(
        '{{resolve:secretsmanager:explorer-agent/redis/auth-token:SecretString:::}}'
      ),
      port: 6379,
      preferredMaintenanceWindow: 'sun:05:00-sun:06:00', // UTC
      snapshotRetentionLimit: 5,
      snapshotWindow: '03:00-05:00', // UTC
      autoMinorVersionUpgrade: true,
      logDeliveryConfigurations: [
        {
          destinationType: 'cloudwatch-logs',
          destinationDetails: {
            cloudWatchLogsDetails: {
              logGroup: '/aws/elasticache/explorer-agent-redis',
            },
          },
          logFormat: 'json',
          logType: 'slow-log',
        },
      ],
    });

    this.replicationGroup.addDependency(subnetGroup);

    // Outputs
    new cdk.CfnOutput(this, 'RedisPrimaryEndpoint', {
      value: this.replicationGroup.attrPrimaryEndPointAddress,
      description: 'Redis primary endpoint address',
      exportName: 'ExplorerAgentRedisPrimaryEndpoint',
    });

    new cdk.CfnOutput(this, 'RedisReaderEndpoint', {
      value: this.replicationGroup.attrReaderEndPointAddress,
      description: 'Redis reader endpoint address',
      exportName: 'ExplorerAgentRedisReaderEndpoint',
    });

    new cdk.CfnOutput(this, 'RedisPort', {
      value: this.replicationGroup.attrPrimaryEndPointPort,
      description: 'Redis port',
      exportName: 'ExplorerAgentRedisPort',
    });
  }
}
