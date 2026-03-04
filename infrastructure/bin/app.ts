#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { NetworkStack } from '../lib/stacks/network-stack';
import { SecurityStack } from '../lib/stacks/security-stack';
import { DatabaseStack } from '../lib/stacks/database-stack';
import { DatabaseSchemaStack } from '../lib/stacks/database-schema-stack';
import { CacheStack } from '../lib/stacks/cache-stack';
import { StorageStack } from '../lib/stacks/storage-stack';
import { MonitoringStack } from '../lib/stacks/monitoring-stack';

const app = new cdk.App();

// Environment configuration
const env = {
  account: process.env.CDK_DEFAULT_ACCOUNT,
  region: process.env.CDK_DEFAULT_REGION || 'us-east-1',
};

// Network Stack - VPC, subnets, security groups
const networkStack = new NetworkStack(app, 'ExplorerAgentNetworkStack', {
  env,
  description: 'Network infrastructure for explorer agent platform',
});

// Security Stack - KMS, Secrets Manager
const securityStack = new SecurityStack(app, 'ExplorerAgentSecurityStack', {
  env,
  description: 'Security infrastructure including KMS and Secrets Manager',
});

// Database Stack - Aurora Serverless v2 PostgreSQL
const databaseStack = new DatabaseStack(app, 'ExplorerAgentDatabaseStack', {
  env,
  description: 'Aurora Serverless v2 PostgreSQL database',
  vpc: networkStack.vpc,
  databaseSecurityGroup: networkStack.databaseSecurityGroup,
  kmsKey: securityStack.kmsKey,
});

// Database Schema Stack - Automated schema deployment
const databaseSchemaStack = new DatabaseSchemaStack(app, 'ExplorerAgentDatabaseSchemaStack', {
  env,
  description: 'Database schema deployment with automated migrations',
  vpc: networkStack.vpc,
  lambdaSecurityGroup: networkStack.lambdaSecurityGroup,
  databaseSecurityGroup: networkStack.databaseSecurityGroup,
  databaseCredentialsSecret: databaseStack.databaseCredentialsSecret,
  databaseEndpoint: databaseStack.cluster.clusterEndpoint.hostname,
});

// Cache Stack - ElastiCache Redis
const cacheStack = new CacheStack(app, 'ExplorerAgentCacheStack', {
  env,
  description: 'ElastiCache Redis cluster for inventory locking',
  vpc: networkStack.vpc,
  cacheSecurityGroup: networkStack.cacheSecurityGroup,
  kmsKey: securityStack.kmsKey,
});

// Storage Stack - S3, CloudFront
const storageStack = new StorageStack(app, 'ExplorerAgentStorageStack', {
  env,
  description: 'S3 buckets and CloudFront distribution',
  kmsKey: securityStack.kmsKey,
});

// Monitoring Stack - CloudWatch, X-Ray
const monitoringStack = new MonitoringStack(app, 'ExplorerAgentMonitoringStack', {
  env,
  description: 'CloudWatch log groups and X-Ray tracing',
});

// Tags
cdk.Tags.of(app).add('Project', 'ExplorerAgentPlatform');
cdk.Tags.of(app).add('ManagedBy', 'CDK');
