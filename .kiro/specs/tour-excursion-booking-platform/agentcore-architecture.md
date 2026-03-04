# AgentCore Architecture - Tour Excursion Booking Platform

## Overview

The AgentCore service is a **decoupled, standalone AI service** that provides conversational AI capabilities for trip planning, itinerary management, and personalized recommendations. It operates independently from the main application backend and integrates through a dedicated API Gateway.

## Architecture Principles

1. **Decoupled Design**: AgentCore runs as a separate service with its own API Gateway, Lambda functions, and data stores
2. **AWS Best Practices**: Follows AWS Bedrock AgentCore runtime patterns and recommendations
3. **Security First**: Implements Guardrails for PII/PCI compliance before any LLM interaction
4. **Memory Management**: Structured short-term and long-term memory strategies
5. **MCP Integration**: Model Context Protocol for external tool access with guardrail filtering

## High-Level Architecture

```
User → API Gateway (Main) → AgentCore Gateway API → AgentCore Runtime
                                                    ↓
                                    ┌───────────────┴──────────────┐
                                    │                              │
                              Guardrails                    Prompt Management
                                    │                              │
                                    └───────────────┬──────────────┘
                                                    ↓
                                            Agentic Layer
                                                    ↓
                                    ┌───────────────┴──────────────┐
                                    │                              │
                              LLM Gateway API              MCP Clients
                                    │                              │
                              AWS Bedrock                  External Tools
                                    │
                                    ↓
                            Memory Management
                                    │
                    ┌───────────────┴───────────────┐
                    │                               │
            Short-Term Memory              Long-Term Memory
            (Aurora PostgreSQL)            (Aurora PostgreSQL + S3)
```

## Components

### 1. AgentCore Gateway API

**Purpose**: Dedicated API Gateway for AgentCore service

**AWS Services**:
- API Gateway (REST API)
- Lambda Authorizer (validates JWT from main Cognito)
- CloudWatch Logs

**Responsibilities**:
- Route requests to AgentCore Lambda functions
- Authenticate requests using JWT tokens from main application
- Rate limiting (10 requests/minute per user)
- Request/response logging for audit

**Endpoints**:
- `POST /agentcore/conversations` - Start new conversation
- `POST /agentcore/conversations/{conversationId}/messages` - Send message
- `GET /agentcore/conversations/{conversationId}` - Get conversation history
- `POST /agentcore/itineraries/generate` - AI-generated itinerary
- `GET /agentcore/health` - Health check

### 2. AgentCore Runtime (AWS Bedrock)

**Purpose**: Core AI inference engine powered by AWS Bedrock

**AWS Services**:
- AWS Bedrock (Claude 3 Sonnet/Haiku)
- AWS Bedrock AgentCore Runtime

**Configuration**:
- Foundation Model: Claude 3 Sonnet (primary), Claude 3 Haiku (fallback)
- Temperature: 0.7 for creative responses
- Max Tokens: 4096
- Streaming: Enabled for real-time responses

**Responsibilities**:
- Natural language understanding
- Intent classification
- Response generation
- Context management

### 3. Guardrails (AWS Bedrock Guardrails)

**Purpose**: Content filtering and compliance enforcement

**AWS Services**:
- AWS Bedrock Guardrails

**Guardrail Policies**:

#### Sensitive Information Filters (PII/PCI)
- **Credit Card Numbers**: BLOCK (PCI compliance)
- **SSN/Tax IDs**: BLOCK
- **Email Addresses**: MASK with `{EMAIL}`
- **Phone Numbers**: MASK with `{PHONE}`
- **Passport Numbers**: MASK with `{PASSPORT}`
- **Bank Account Numbers**: BLOCK
- **Driver's License**: MASK with `{LICENSE}`
- **Names**: MASK with `{NAME}` (in responses only)
- **Addresses**: MASK with `{ADDRESS}` (in responses only)

#### Content Filters
- **Hate Speech**: BLOCK (high confidence)
- **Insults**: BLOCK (high confidence)
- **Sexual Content**: BLOCK (high confidence)
- **Violence**: BLOCK (medium confidence)
- **Misconduct**: BLOCK (medium confidence)

#### Denied Topics
- **Financial Advice**: "I cannot provide financial advice"
- **Medical Advice**: "I cannot provide medical advice"
- **Legal Advice**: "I cannot provide legal advice"
- **Political Opinions**: "I focus on travel planning"

#### Prompt Attack Detection
- **Jailbreak Attempts**: BLOCK
- **Prompt Injection**: BLOCK

**MCP Filter Integration**:
- All MCP tool calls flow through guardrails
- Tool responses are filtered before returning to user
- Sensitive data in tool responses is masked

**Implementation**:
```go
// Guardrail configuration
type GuardrailConfig struct {
    GuardrailID      string
    GuardrailVersion string
    Trace            bool
}

// Apply guardrail to input
func ApplyInputGuardrail(input string, config GuardrailConfig) (*GuardrailResult, error)

// Apply guardrail to output
func ApplyOutputGuardrail(output string, config GuardrailConfig) (*GuardrailResult, error)
```

### 4. Prompt Management (AWS Bedrock)

**Purpose**: Centralized prompt templates for different use cases

**AWS Services**:
- AWS Bedrock Prompt Management
- S3 (prompt version storage)

**Prompt Categories**:

#### Trip Planning Prompts
- `trip-planner-v1`: General trip planning assistant
- `cruise-planner-v1`: Cruise-specific itinerary planning
- `flight-planner-v1`: Flight arrival itinerary planning

#### Recommendation Prompts
- `excursion-recommender-v1`: Excursion recommendations
- `dining-recommender-v1`: Restaurant recommendations
- `shopping-recommender-v1`: Shopping location recommendations

#### Conflict Resolution Prompts
- `conflict-detector-v1`: Identify scheduling conflicts
- `conflict-resolver-v1`: Suggest conflict resolutions

#### Conversation Prompts
- `greeting-v1`: Initial conversation greeting
- `clarification-v1`: Ask clarifying questions
- `summary-v1`: Summarize conversation

**Prompt Structure**:
```
System Prompt:
You are a helpful travel planning assistant for ShoreEx platform.
Your role is to help users plan their trips, book excursions, and discover dining and shopping experiences.

Context:
- User Segment: {user_segment}
- Destination: {destination}
- Travel Dates: {start_date} to {end_date}
- Time Window: {time_window}
- Budget Level: {budget_level}

Guidelines:
1. Always prioritize user safety and satisfaction
2. Respect time windows for cruise/flight schedules
3. Suggest activities within budget constraints
4. Detect and resolve scheduling conflicts
5. Provide personalized recommendations based on user preferences

Constraints:
- Do not provide financial, medical, or legal advice
- Do not share personal information about other users
- Do not make bookings without explicit user confirmation
```

### 5. LLM Gateway API

**Purpose**: Abstraction layer for LLM provider interactions

**AWS Services**:
- Lambda (Go runtime)
- AWS Bedrock SDK

**Responsibilities**:
- Model selection (Claude 3 Sonnet/Haiku)
- Request formatting
- Response parsing
- Error handling and retries
- Token usage tracking
- Cost optimization

**Features**:
- **Model Fallback**: Sonnet → Haiku on rate limits
- **Streaming Support**: Server-sent events for real-time responses
- **Context Window Management**: Automatic truncation at 100K tokens
- **Caching**: Prompt caching for repeated system prompts (50% cost reduction)

**Implementation**:
```go
type LLMGatewayRequest struct {
    Model            string
    SystemPrompt     string
    Messages         []Message
    Temperature      float64
    MaxTokens        int
    Stream           bool
    GuardrailConfig  *GuardrailConfig
}

type LLMGatewayResponse struct {
    Content          string
    TokensUsed       int
    Model            string
    FinishReason     string
    GuardrailAction  string
}
```

### 6. MCP Clients (Model Context Protocol)

**Purpose**: Integration with external tools and data sources

**MCP Tools**:

#### Booking Tools
- `search_excursions`: Search available excursions
- `get_excursion_details`: Get detailed excursion information
- `check_availability`: Check real-time availability
- `create_booking`: Create booking (requires confirmation)

#### Dining Tools
- `search_restaurants`: Search restaurants by cuisine/location
- `get_restaurant_details`: Get restaurant details
- `get_dining_recommendations`: Get personalized dining suggestions

#### Shopping Tools
- `search_shopping_locations`: Search shopping venues
- `get_local_products`: Get local product recommendations

#### Itinerary Tools
- `create_itinerary`: Create new itinerary
- `add_itinerary_item`: Add item to itinerary
- `detect_conflicts`: Check for scheduling conflicts
- `resolve_conflicts`: Suggest conflict resolutions

#### User Tools
- `get_user_preferences`: Get user preferences and history
- `get_user_bookings`: Get user's past bookings
- `get_user_segment`: Get user segment classification

**Guardrail Integration**:
- All tool inputs are filtered through guardrails
- Tool responses are filtered before returning to LLM
- Sensitive data in tool responses is masked
- Failed guardrail checks block tool execution

**Implementation**:
```go
type MCPClient struct {
    ToolName        string
    GuardrailConfig GuardrailConfig
}

func (c *MCPClient) ExecuteTool(input map[string]interface{}) (*ToolResponse, error) {
    // 1. Apply input guardrail
    filteredInput, err := ApplyInputGuardrail(input, c.GuardrailConfig)
    if err != nil || filteredInput.Action == "BLOCKED" {
        return nil, errors.New("input blocked by guardrail")
    }
    
    // 2. Execute tool
    response, err := c.callTool(filteredInput.Content)
    if err != nil {
        return nil, err
    }
    
    // 3. Apply output guardrail
    filteredOutput, err := ApplyOutputGuardrail(response, c.GuardrailConfig)
    if err != nil || filteredOutput.Action == "BLOCKED" {
        return nil, errors.New("output blocked by guardrail")
    }
    
    return filteredOutput, nil
}
```

### 7. Memory Management

**Purpose**: Structured memory strategies for conversation persistence and learning

**AWS Services**:
- Aurora Serverless v2 PostgreSQL (primary storage)
- S3 (long-term memory snapshots)

#### Short-Term Memory (STM)

**Storage**: Aurora PostgreSQL with TTL cleanup via pg_cron

**Tables**:
```sql
-- Conversation sessions
CREATE TABLE agentcore_conversations (
    conversation_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    actor_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    metadata JSONB,
    INDEX idx_user_id (user_id),
    INDEX idx_expires_at (expires_at)
);

-- Conversation messages
CREATE TABLE agentcore_messages (
    message_id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES agentcore_conversations(conversation_id),
    role VARCHAR(50) NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    tokens_used INT,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB,
    INDEX idx_conversation_id (conversation_id),
    INDEX idx_timestamp (timestamp)
);

-- User conversation state
CREATE TABLE agentcore_conversation_state (
    state_id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES agentcore_conversations(conversation_id),
    state_key VARCHAR(255) NOT NULL,
    state_value JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(conversation_id, state_key),
    INDEX idx_conversation_id (conversation_id)
);
```

**TTL Cleanup**:
```sql
-- Cleanup expired conversations (runs daily)
CREATE EXTENSION IF NOT EXISTS pg_cron;

SELECT cron.schedule(
    'cleanup-expired-conversations',
    '0 2 * * *', -- 2 AM daily
    $$
    DELETE FROM agentcore_conversations 
    WHERE expires_at < NOW() - INTERVAL '7 days';
    $$
);
```

**Retention**: 24 hours active, 7 days grace period

**Data Stored**:
- Conversation messages (last 50 messages)
- Current conversation state (destination, dates, preferences)
- Pending actions (bookings awaiting confirmation)
- Semantic memory (extracted facts from current session)

#### Long-Term Memory (LTM)

**Storage**: Aurora PostgreSQL + S3

**Tables**:
```sql
-- User preferences (learned over time)
CREATE TABLE agentcore_user_preferences (
    preference_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    actor_id VARCHAR(255) NOT NULL,
    preference_type VARCHAR(100) NOT NULL, -- 'cuisine', 'activity', 'budget', 'difficulty'
    preference_value TEXT NOT NULL,
    confidence_score FLOAT NOT NULL DEFAULT 0.5,
    learned_from_conversation_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    INDEX idx_user_id (user_id),
    INDEX idx_preference_type (preference_type)
);

-- Semantic memory (facts about user)
CREATE TABLE agentcore_semantic_memory (
    memory_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    actor_id VARCHAR(255) NOT NULL,
    fact_type VARCHAR(100) NOT NULL, -- 'dietary', 'accessibility', 'travel_style', 'companion'
    fact_content TEXT NOT NULL,
    confidence_score FLOAT NOT NULL DEFAULT 0.5,
    source_conversation_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    INDEX idx_user_id (user_id),
    INDEX idx_fact_type (fact_type)
);

-- Conversation summaries
CREATE TABLE agentcore_conversation_summaries (
    summary_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    actor_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    conversation_id UUID NOT NULL,
    summary_text TEXT NOT NULL,
    key_decisions JSONB, -- Important decisions made
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    INDEX idx_user_id (user_id),
    INDEX idx_session_id (session_id)
);

-- Successful itineraries (for learning)
CREATE TABLE agentcore_successful_itineraries (
    itinerary_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    destination_id UUID NOT NULL,
    user_segment VARCHAR(50) NOT NULL,
    itinerary_data JSONB NOT NULL,
    user_rating INT, -- 1-5 stars
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    s3_snapshot_key VARCHAR(500), -- S3 key for detailed snapshot
    INDEX idx_user_id (user_id),
    INDEX idx_destination_id (destination_id),
    INDEX idx_user_segment (user_segment)
);
```

**S3 Storage**:
- Path: `s3://agentcore-memory/{user_id}/itineraries/{itinerary_id}.json`
- Retention: Indefinite (lifecycle policy to Glacier after 1 year)
- Encryption: SSE-KMS

**Memory Strategies**:

1. **User Preference Memory Strategy**
   - Namespace: `/preferences/{actorId}`
   - Learns user preferences from conversations
   - Confidence scoring (0.0-1.0)
   - Updates existing preferences with new evidence

2. **Semantic Memory Strategy**
   - Namespace: `/facts/{actorId}`
   - Extracts factual information about user
   - Stores dietary restrictions, accessibility needs, travel companions
   - Used for personalization

3. **Summary Memory Strategy**
   - Namespace: `/summaries/{actorId}/{sessionId}`
   - Summarizes completed conversations
   - Stores key decisions and outcomes
   - Used for context in future conversations

**Retrieval Configuration**:
```go
type MemoryRetrievalConfig struct {
    Preferences struct {
        TopK            int     // 5
        RelevanceScore  float64 // 0.7
    }
    Facts struct {
        TopK            int     // 10
        RelevanceScore  float64 // 0.3
    }
    Summaries struct {
        TopK            int     // 5
        RelevanceScore  float64 // 0.5
    }
}
```

### 8. Conversation State Management

**Purpose**: Track conversation context and user intent

**State Components**:

#### Current Intent
- `trip_planning`: Planning a new trip
- `booking_excursion`: Booking specific excursion
- `finding_restaurants`: Looking for dining options
- `finding_shopping`: Looking for shopping locations
- `managing_itinerary`: Modifying existing itinerary
- `asking_question`: General question about destination/service

#### Context Variables
```json
{
  "destination_id": "uuid",
  "destination_name": "Cozumel, Mexico",
  "travel_dates": {
    "start": "2026-06-15",
    "end": "2026-06-15"
  },
  "time_window": {
    "earliest_start": "2026-06-15T09:00:00Z",
    "latest_end": "2026-06-15T17:00:00Z"
  },
  "cruise_info": {
    "cruise_line": "Royal Caribbean",
    "ship_name": "Harmony of the Seas",
    "port_call_id": "uuid"
  },
  "group_composition": {
    "adults": 2,
    "children": 1,
    "infants": 0
  },
  "budget_level": "moderate",
  "user_segment": "family",
  "preferences": {
    "activity_types": ["beach", "snorkeling", "cultural"],
    "dietary_restrictions": ["vegetarian"],
    "accessibility_needs": []
  },
  "pending_actions": [
    {
      "action_type": "booking_confirmation",
      "excursion_id": "uuid",
      "requires_confirmation": true
    }
  ]
}
```

## Lambda Functions

### 1. AgentCore Orchestrator Lambda

**Runtime**: Go 1.x
**Memory**: 1024 MB
**Timeout**: 60 seconds

**Responsibilities**:
- Receive user messages
- Load conversation context from memory
- Apply input guardrails
- Call LLM Gateway API
- Apply output guardrails
- Update conversation state
- Save messages to memory
- Return response to user

### 2. Memory Manager Lambda

**Runtime**: Go 1.x
**Memory**: 512 MB
**Timeout**: 30 seconds

**Responsibilities**:
- Load short-term memory (last 50 messages)
- Load long-term memory (preferences, facts, summaries)
- Save new messages
- Update conversation state
- Extract and store preferences
- Extract and store semantic facts
- Generate conversation summaries

### 3. MCP Tool Executor Lambda

**Runtime**: Go 1.x
**Memory**: 1024 MB
**Timeout**: 30 seconds

**Responsibilities**:
- Execute MCP tool calls
- Apply guardrails to tool inputs/outputs
- Call main application APIs
- Format tool responses for LLM
- Handle tool errors gracefully

### 4. Itinerary Generator Lambda

**Runtime**: Go 1.x
**Memory**: 2048 MB
**Timeout**: 120 seconds

**Responsibilities**:
- Generate AI-powered itineraries
- Query available excursions, restaurants, shopping
- Optimize for time windows and preferences
- Detect and resolve conflicts
- Calculate total costs
- Save generated itinerary

### 5. Conversation Cleanup Lambda

**Runtime**: Go 1.x
**Memory**: 256 MB
**Timeout**: 300 seconds
**Trigger**: EventBridge (daily at 2 AM)

**Responsibilities**:
- Archive expired conversations to S3
- Delete conversations older than 7 days
- Generate conversation summaries for LTM
- Update user preference confidence scores

## Data Flow

### User Message Flow

```
1. User sends message via main API Gateway
2. Main API Gateway routes to AgentCore Gateway API
3. AgentCore Gateway validates JWT token
4. AgentCore Orchestrator Lambda receives request
5. Memory Manager loads conversation context
6. Input Guardrail filters user message
7. LLM Gateway formats request with prompt template
8. AWS Bedrock generates response
9. Output Guardrail filters response
10. Memory Manager saves message and updates state
11. Response returned to user
```

### MCP Tool Call Flow

```
1. LLM decides to call MCP tool
2. MCP Tool Executor receives tool call
3. Input Guardrail filters tool parameters
4. Tool Executor calls main application API
5. Tool Executor receives response
6. Output Guardrail filters tool response
7. Filtered response returned to LLM
8. LLM incorporates tool result in response
```

### Memory Update Flow

```
1. Conversation completes
2. Memory Manager extracts preferences from messages
3. Memory Manager extracts semantic facts
4. Memory Manager generates conversation summary
5. Preferences saved to agentcore_user_preferences
6. Facts saved to agentcore_semantic_memory
7. Summary saved to agentcore_conversation_summaries
8. If itinerary created, save to agentcore_successful_itineraries
9. Snapshot saved to S3
```

## Security & Compliance

### PII/PCI Compliance

1. **Input Filtering**: All user inputs filtered through guardrails
2. **Output Masking**: PII in responses masked with placeholders
3. **Credit Card Blocking**: Credit card numbers blocked immediately
4. **Audit Logging**: All guardrail actions logged to CloudWatch
5. **Encryption**: All data encrypted at rest (KMS) and in transit (TLS 1.3)

### Access Control

1. **JWT Authentication**: All requests require valid JWT from main Cognito
2. **User Isolation**: Users can only access their own conversations
3. **IAM Roles**: Least privilege for Lambda execution roles
4. **VPC Isolation**: AgentCore Lambdas run in private subnets

### Monitoring

1. **CloudWatch Metrics**: Token usage, latency, error rates
2. **CloudWatch Logs**: All requests/responses logged
3. **X-Ray Tracing**: End-to-end request tracing
4. **Guardrail Metrics**: Blocked requests, masked PII counts
5. **Cost Tracking**: Token usage and costs per user

## Cost Optimization

1. **Prompt Caching**: Cache system prompts (50% cost reduction)
2. **Model Selection**: Use Haiku for simple queries, Sonnet for complex
3. **Context Window Management**: Limit to last 50 messages
4. **Batch Processing**: Batch memory updates
5. **S3 Lifecycle**: Move old snapshots to Glacier

## Deployment

### Infrastructure (AWS CDK)

```typescript
// lib/stacks/agentcore-stack.ts
export class AgentCoreStack extends Stack {
  constructor(scope: Construct, id: string, props: StackProps) {
    super(scope, id, props);
    
    // AgentCore Gateway API
    const agentCoreApi = new apigateway.RestApi(this, 'AgentCoreAPI', {
      restApiName: 'AgentCore Gateway',
      description: 'Dedicated API for AgentCore service',
    });
    
    // Guardrails
    const guardrail = new bedrock.CfnGuardrail(this, 'AgentCoreGuardrail', {
      name: 'agentcore-guardrail',
      blockedInputMessaging: 'I cannot process that request.',
      blockedOutputsMessaging: 'I cannot provide that information.',
      sensitiveInformationPolicyConfig: {
        piiEntitiesConfig: [
          { type: 'CREDIT_DEBIT_CARD_NUMBER', action: 'BLOCK' },
          { type: 'EMAIL', action: 'MASK' },
          { type: 'PHONE', action: 'MASK' },
          { type: 'NAME', action: 'MASK' },
          { type: 'ADDRESS', action: 'MASK' },
        ],
      },
      contentPolicyConfig: {
        filtersConfig: [
          { type: 'HATE', inputStrength: 'HIGH', outputStrength: 'HIGH' },
          { type: 'INSULTS', inputStrength: 'HIGH', outputStrength: 'HIGH' },
          { type: 'SEXUAL', inputStrength: 'HIGH', outputStrength: 'HIGH' },
          { type: 'VIOLENCE', inputStrength: 'MEDIUM', outputStrength: 'MEDIUM' },
        ],
      },
    });
    
    // Lambda functions
    const orchestratorLambda = new lambda.Function(this, 'OrchestratorLambda', {
      runtime: lambda.Runtime.GO_1_X,
      handler: 'main',
      code: lambda.Code.fromAsset('dist/agentcore-orchestrator'),
      memorySize: 1024,
      timeout: Duration.seconds(60),
      environment: {
        GUARDRAIL_ID: guardrail.attrGuardrailId,
        GUARDRAIL_VERSION: 'DRAFT',
      },
    });
    
    // ... more resources
  }
}
```

## Testing Strategy

1. **Unit Tests**: Test individual Lambda functions
2. **Integration Tests**: Test end-to-end conversation flows
3. **Guardrail Tests**: Verify PII/PCI filtering
4. **Load Tests**: Test concurrent conversations
5. **Cost Tests**: Monitor token usage and costs

## Monitoring & Alerts

### CloudWatch Alarms

1. **High Error Rate**: > 5% error rate
2. **High Latency**: > 5 seconds p99 latency
3. **Guardrail Blocks**: > 100 blocks/hour
4. **Token Usage**: > 1M tokens/day
5. **Cost Threshold**: > $500/day

### Dashboards

1. **AgentCore Overview**: Conversations, messages, token usage
2. **Guardrail Metrics**: Blocks, masks, denied topics
3. **Memory Metrics**: STM/LTM sizes, retrieval latency
4. **Cost Dashboard**: Token costs by model, user segment

## Future Enhancements

1. **Multi-Modal Support**: Image understanding for destination photos
2. **Voice Integration**: Amazon Polly for voice responses
3. **Proactive Suggestions**: EventBridge-triggered recommendations
4. **A/B Testing**: Test different prompts and models
5. **Fine-Tuning**: Custom model fine-tuning on successful conversations
