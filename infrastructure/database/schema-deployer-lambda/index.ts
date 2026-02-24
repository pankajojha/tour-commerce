import { Handler } from 'aws-lambda';
import { SecretsManagerClient, GetSecretValueCommand } from '@aws-sdk/client-secrets-manager';
import { Client } from 'pg';
import * as fs from 'fs';
import * as path from 'path';

interface DatabaseCredentials {
  username: string;
  password: string;
  host: string;
  port: number;
  dbname: string;
}

interface MigrationRecord {
  version: string;
  description: string;
  applied_at: Date;
}

const secretsManager = new SecretsManagerClient({});

/**
 * Retrieves database credentials from AWS Secrets Manager
 */
async function getDatabaseCredentials(secretArn: string): Promise<DatabaseCredentials> {
  const command = new GetSecretValueCommand({ SecretId: secretArn });
  const response = await secretsManager.send(command);
  
  if (!response.SecretString) {
    throw new Error('Secret value is empty');
  }
  
  return JSON.parse(response.SecretString);
}

/**
 * Creates the schema_migrations table if it doesn't exist
 */
async function ensureMigrationsTable(client: Client): Promise<void> {
  const createTableSQL = `
    CREATE TABLE IF NOT EXISTS schema_migrations (
      version VARCHAR(50) PRIMARY KEY,
      description VARCHAR(255) NOT NULL,
      applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      checksum VARCHAR(64) NOT NULL
    );
  `;
  
  await client.query(createTableSQL);
  console.log('Ensured schema_migrations table exists');
}

/**
 * Gets list of applied migrations from database
 */
async function getAppliedMigrations(client: Client): Promise<Set<string>> {
  const result = await client.query<MigrationRecord>(
    'SELECT version FROM schema_migrations ORDER BY version'
  );
  
  return new Set(result.rows.map(row => row.version));
}

/**
 * Calculates checksum for migration file content
 */
function calculateChecksum(content: string): string {
  const crypto = require('crypto');
  return crypto.createHash('sha256').update(content).digest('hex');
}

/**
 * Parses migration filename to extract version and description
 * Format: V001__initial_schema.sql -> { version: '001', description: 'initial_schema' }
 */
function parseMigrationFilename(filename: string): { version: string; description: string } | null {
  const match = filename.match(/^V(\d+)__(.+)\.sql$/);
  if (!match) {
    return null;
  }
  
  return {
    version: match[1],
    description: match[2].replace(/_/g, ' '),
  };
}

/**
 * Reads all migration files from the migrations directory
 */
function getMigrationFiles(migrationsDir: string): Array<{ filename: string; version: string; description: string; content: string }> {
  const files = fs.readdirSync(migrationsDir)
    .filter(f => f.endsWith('.sql'))
    .sort();
  
  const migrations = [];
  
  for (const filename of files) {
    const parsed = parseMigrationFilename(filename);
    if (!parsed) {
      console.warn(`Skipping invalid migration filename: ${filename}`);
      continue;
    }
    
    const content = fs.readFileSync(path.join(migrationsDir, filename), 'utf-8');
    migrations.push({
      filename,
      version: parsed.version,
      description: parsed.description,
      content,
    });
  }
  
  return migrations;
}

/**
 * Applies a single migration to the database
 */
async function applyMigration(
  client: Client,
  version: string,
  description: string,
  content: string
): Promise<void> {
  console.log(`Applying migration ${version}: ${description}`);
  
  const checksum = calculateChecksum(content);
  
  // Start transaction
  await client.query('BEGIN');
  
  try {
    // Execute migration SQL
    await client.query(content);
    
    // Record migration in schema_migrations table
    await client.query(
      'INSERT INTO schema_migrations (version, description, checksum) VALUES ($1, $2, $3)',
      [version, description, checksum]
    );
    
    // Commit transaction
    await client.query('COMMIT');
    
    console.log(`Successfully applied migration ${version}`);
  } catch (error) {
    // Rollback on error
    await client.query('ROLLBACK');
    console.error(`Failed to apply migration ${version}:`, error);
    throw error;
  }
}

/**
 * Main Lambda handler
 */
export const handler: Handler = async (event) => {
  console.log('Schema Deployer Lambda invoked', JSON.stringify(event, null, 2));
  
  const secretArn = process.env.DB_SECRET_ARN;
  const migrationsDir = process.env.MIGRATIONS_DIR || '/var/task/migrations';
  
  if (!secretArn) {
    throw new Error('DB_SECRET_ARN environment variable is required');
  }
  
  let client: Client | null = null;
  
  try {
    // Get database credentials
    console.log('Retrieving database credentials from Secrets Manager');
    const credentials = await getDatabaseCredentials(secretArn);
    
    // Connect to database
    console.log(`Connecting to database at ${credentials.host}:${credentials.port}`);
    client = new Client({
      host: credentials.host,
      port: credentials.port,
      database: credentials.dbname,
      user: credentials.username,
      password: credentials.password,
      ssl: {
        rejectUnauthorized: false, // Aurora uses self-signed certificates
      },
    });
    
    await client.connect();
    console.log('Connected to database successfully');
    
    // Ensure migrations table exists
    await ensureMigrationsTable(client);
    
    // Get applied migrations
    const appliedMigrations = await getAppliedMigrations(client);
    console.log(`Found ${appliedMigrations.size} applied migrations`);
    
    // Get migration files
    const migrationFiles = getMigrationFiles(migrationsDir);
    console.log(`Found ${migrationFiles.length} migration files`);
    
    // Apply pending migrations
    let appliedCount = 0;
    for (const migration of migrationFiles) {
      if (appliedMigrations.has(migration.version)) {
        console.log(`Skipping already applied migration ${migration.version}`);
        continue;
      }
      
      await applyMigration(client, migration.version, migration.description, migration.content);
      appliedCount++;
    }
    
    console.log(`Applied ${appliedCount} new migrations`);
    
    return {
      statusCode: 200,
      body: JSON.stringify({
        message: 'Schema deployment completed successfully',
        appliedMigrations: appliedCount,
        totalMigrations: migrationFiles.length,
      }),
    };
  } catch (error) {
    console.error('Schema deployment failed:', error);
    throw error;
  } finally {
    if (client) {
      await client.end();
      console.log('Database connection closed');
    }
  }
};
