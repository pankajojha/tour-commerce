# Implementation Plan: Tour Excursion Booking Platform

## Overview

This implementation plan breaks down the AWS-based tour excursion booking platform into discrete, sequential coding tasks. The platform enables travelers to discover and book shore excursions with intelligent time-window filtering for cruise passengers and flight arrivals, real-time inventory management, dynamic pricing, and secure payment processing. The architecture leverages AWS serverless services including Lambda, API Gateway, Cognito, Aurora Serverless v2 PostgreSQL, ElastiCache Redis, and OpenSearch.

**Technology Stack:**
- Backend: Go (Golang) for all Lambda functions
- Frontend: React with TypeScript for UI components
- Infrastructure: AWS CDK with TypeScript
- Testing: Go testing framework with gopter or rapid for property-based tests

## Tasks

- [x] 1. Set up AWS infrastructure foundation and core services
  - [x] 1.1 Create core AWS infrastructure
    - Create AWS CDK project structure with TypeScript
    - Define VPC with public/private subnets, NAT Gateway, and security groups
    - Set up Aurora Serverless v2 PostgreSQL cluster with auto-scaling
    - Configure ElastiCache Redis cluster with encryption at rest
    - Configure S3 buckets for vouchers and static assets with encryption
    - Set up CloudFront CDN distribution
    - Set up Secrets Manager for API keys and database credentials
    - Configure KMS keys for encryption
    - Set up CloudWatch log groups and X-Ray tracing
    - _Requirements: NFR-1, NFR-2, NFR-3, NFR-4, 21.1, 21.2, 21.3_
  
  - [ ]* 1.2 Configure AWS WAF (optional - future backlog)
    - Configure AWS WAF with SQL injection and XSS protection rules
    - _Requirements: 22.4, 22.5_

- [x] 2. Create initial database schema
  - Create initial database schema SQL scripts
  - Create tables: users, destinations, excursions, excursion_schedules, suppliers, cruise_port_calls, flight_arrivals, bookings, payments, refunds, reviews, categories, inventory_records, pricing_configurations, sessions, passwordless_tokens
  - Create sessions table with JSONB data column, expires_at timestamp, and indexes
  - Create passwordless_tokens table (token_id, user_email, token_hash, token_type, expires_at, attempts, created_at)
  - Configure pg_cron extension for automated session cleanup and expired token cleanup
  - Create indexes on frequently queried fields (destination_id, date, cruise_line, excursion_id, session expires_at, user_email, token_hash)
  - Set up foreign key constraints and cascading rules
  - Configure database connection pooling with RDS Proxy
  - _Requirements: 1.1, 1.6, 2.1, 2.2, 3.1, 4.1, 5.1, 7.1, 8.1, 14.1_


- [x] 3. Implement Authentication Service with Cognito integration
  - [x] 3.1 Configure Amazon Cognito User Pool with email and phone attributes
    - Set up user pool with password policy (8+ chars, complexity requirements)
    - Configure email verification workflow
    - Enable passwordless authentication with custom auth flow
    - Configure magic link authentication (email-based OTP)
    - Set up OAuth 2.0 app clients for web and mobile
    - Configure identity providers for Facebook, Google, Instagram, TikTok, LinkedIn, Microsoft
    - Store OAuth client secrets in Secrets Manager
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 1.10_
  
  - [x] 3.2 Implement Authentication Lambda functions in Go
    - Create signup Lambda with email/password registration (Go)
    - Create login Lambda with credential validation (Go)
    - Create social login Lambda with OAuth 2.0 flow (Go)
    - Create passwordless login Lambda with magic link generation (Go)
    - Create magic link verification Lambda (Go)
    - Create token refresh Lambda (Go)
    - Create password reset Lambda with email verification (Go)
    - Implement Cognito Lambda triggers (pre-signup, post-confirmation, pre-token generation) (Go)
    - Implement custom auth challenge Lambda for passwordless flow (Go)
    - Create user profile in RDS on successful authentication
    - _Requirements: 1.1, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9_
  
  - [x] 3.3 Implement passwordless authentication flow
    - Generate secure one-time tokens (6-digit code or magic link)
    - Store tokens in Aurora PostgreSQL with 10-minute expiration
    - Send magic link via Amazon SES
    - Implement token validation with rate limiting (3 attempts)
    - Generate JWT tokens on successful verification
    - Log all passwordless authentication attempts
    - _Requirements: 1.1, 1.3, 1.4_
  
  - [ ] 3.4 Manual validation of authentication flows
    - **AWS Console Testing:**
      - Create test users in Cognito console
      - Verify email verification workflow
      - Test user pool password policy enforcement
      - Verify OAuth provider configuration
    - **API Testing with Postman/Insomnia:**
      - Test POST /auth/signup with valid/invalid emails
      - Test POST /auth/login and verify JWT token structure
      - Test POST /auth/login with wrong password (should fail)
      - Test POST /auth/refresh with valid refresh token
      - Test POST /auth/passwordless/request (magic link generation)
      - Test POST /auth/passwordless/verify with token
      - Test POST /auth/password-reset/request
      - Test social login endpoints for each provider
      - Save all requests as collection for regression testing
    - **AWS CLI Testing:**
      - Test signup: `aws cognito-idp sign-up --client-id XXX --username test@example.com --password Pass123!`
      - Test login: `aws cognito-idp initiate-auth --client-id XXX --auth-flow USER_PASSWORD_AUTH`
      - Test token refresh: `aws cognito-idp initiate-auth --auth-flow REFRESH_TOKEN_AUTH`
      - Verify JWT token claims with `jwt.io`
    - **Protected Endpoint Testing:**
      - Test API Gateway endpoint without Authorization header (should return 401)
      - Test with valid JWT token in Authorization header (should succeed)
      - Test with expired JWT token (should return 401)
      - Test with malformed JWT token (should return 401)
    - **Email Testing:**
      - Verify email verification emails are sent
      - Verify magic link emails are sent with correct format
      - Verify password reset emails are sent
      - Test magic link expiration (10 minutes)
    - **Rate Limiting Testing:**
      - Attempt passwordless login 4+ times (should block after 3 attempts)
      - Verify rate limit resets after time window
    - **Database Validation:**
      - Query users table to verify user profile creation
      - Query passwordless_tokens table to verify token storage
      - Verify token expiration timestamps are correct
      - Verify tokens are cleaned up after expiration
    - _Requirements: 1.1, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9_
  
  - [ ]* 3.5 Write property test for authentication token generation using gopter or rapid (optional)
    - **Property 13: User Authentication - All authenticated API requests have valid JWT tokens**
    - **Validates: Requirements 1.4, 22.1, 22.2**
  
  - [ ]* 3.6 Write unit tests for authentication service using Go testing (optional - write after manual validation)
    - Test email validation and uniqueness
    - Test password complexity requirements
    - Test social login token exchange
    - Test passwordless magic link generation
    - Test magic link expiration and validation
    - Test JWT token expiration
    - Test rate limiting for passwordless attempts
    - _Requirements: 1.1, 1.4, 1.9_

- [ ] 4. Implement API Gateway and authorization
  - Create REST API in API Gateway with CORS configuration
  - Configure Cognito authorizer for protected endpoints
  - Set up API Gateway request validation with JSON schemas
  - Configure throttling (10,000 requests/sec) and burst limits
  - Set up API Gateway caching with 5-minute TTL for search endpoints
  - Create API Gateway stages (dev, staging, production)
  - _Requirements: 22.1, 22.2, 22.3, 22.9, 26.9, NFR-1_


- [ ] 5. Implement Destination Service
  - [ ] 5.1 Create Destination Lambda functions in Go
    - Implement searchDestinations with OpenSearch full-text search (Go)
    - Implement getDestination with destination details (Go)
    - Implement getDestinationByPortCode for cruise integration (Go)
    - Implement getDestinationByAirportCode for flight integration (Go)
    - Implement listDestinations with pagination (Go)
    - _Requirements: 2.1, 2.2, 3.1, 4.4_
  
  - [ ] 5.2 Set up OpenSearch cluster and indexing
    - Create OpenSearch domain with encryption
    - Define destination index mapping with full-text fields
    - Implement indexing Lambda in Go to sync RDS data to OpenSearch
    - Configure autocomplete with edge n-grams
    - _Requirements: 2.1, 25.1, 25.8_
  
  - [ ]* 5.3 Write unit tests for destination service using Go testing
    - Test search with various filters
    - Test port code and airport code lookups
    - Test pagination
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [ ] 6. Implement Excursion Service with schedule management
  - [ ] 6.1 Create Excursion Lambda functions in Go
    - Implement searchExcursions with OpenSearch and filters (Go)
    - Implement getExcursion with full excursion details (Go)
    - Implement getExcursionSchedule for specific dates (Go)
    - Implement createExcursion and updateExcursion for admin (Go)
    - _Requirements: 2.3, 2.4, 2.5, 2.6, 2.7, 2.8_
  
  - [ ] 6.2 Implement excursion filtering logic
    - Create filterByCategory, filterByPrice, filterByDuration, filterByDifficulty functions
    - Implement multi-filter AND logic
    - Implement facet counting for filter options
    - _Requirements: 2.4, 25.2, 25.3, 25.4, 25.5, 25.6, 25.7_
  
  - [ ] 6.3 Index excursions in OpenSearch
    - Define excursion index mapping with nested schedules
    - Implement indexing Lambda in Go for excursion data
    - Include destination name, categories, supplier name in index
    - _Requirements: 2.3, 25.1, 25.10_
  
  - [ ]* 6.4 Write unit tests for excursion service using Go testing
    - Test search with multiple filters
    - Test schedule retrieval
    - Test blackout date handling
    - _Requirements: 2.3, 2.4, 2.5, 2.6_


- [ ] 7. Implement Cruise Schedule Service with time-window calculations
  - [ ] 7.1 Create Cruise Schedule Lambda functions in Go
    - Implement getPortCall to retrieve cruise port call details (Go)
    - Implement searchPortCalls with filters (cruise line, ship, destination, date range) (Go)
    - Implement calculateTimeWindow with arrival buffer (+1hr) and departure buffer (-1.5hr) (Go)
    - Implement importSchedules for admin to ingest cruise data (Go)
    - Handle timezone-aware date/time parsing using IANA timezones
    - _Requirements: 3.1, 3.2, 3.6, 3.8, 17.3, 17.7_
  
  - [ ] 7.2 Implement cruise excursion eligibility filtering
    - Create getExcursionsForCruise that filters by time window
    - Verify excursion start time >= earliest start (arrival + 1hr)
    - Verify excursion end time <= latest end (departure - 1.5hr)
    - Check excursion operates on the day of week
    - Exclude blackout dates
    - _Requirements: 3.3, 3.4, 3.5_
  
  - [ ]* 7.3 Write property test for time window calculations using gopter or rapid
    - **Property 3: Time Window Validity - Cruise time windows always have positive duration**
    - **Validates: Requirements 3.2, 17.3**
  
  - [ ]* 7.4 Write unit tests for cruise schedule service using Go testing
    - Test time window calculation with various arrival/departure times
    - Test timezone conversions
    - Test excursion filtering with time windows
    - Test day of week validation
    - _Requirements: 3.2, 3.3, 3.4, 3.5, 3.8_

- [ ] 8. Implement Flight Schedule Service with transfer buffer
  - [ ] 8.1 Create Flight Schedule Lambda functions in Go
    - Implement getFlightArrival to retrieve flight details (Go)
    - Implement searchFlightArrivals with filters (Go)
    - Implement calculateEarliestStart with 2-hour transfer buffer (Go)
    - Implement importFlightSchedules to ingest flight data from external APIs (Go)
    - Handle timezone-aware date/time parsing
    - _Requirements: 4.1, 4.2, 4.5, 4.6, 4.7, 17.4, 17.8_
  
  - [ ] 8.2 Implement flight excursion eligibility filtering
    - Create getExcursionsForFlight that filters by earliest start time
    - Verify excursion start time >= arrival time + 2 hours
    - Match excursions by destination airport code
    - _Requirements: 4.3, 4.4_
  
  - [ ]* 8.3 Write unit tests for flight schedule service using Go testing
    - Test earliest start calculation
    - Test excursion filtering by airport code
    - Test timezone handling
    - _Requirements: 4.1, 4.2, 4.3, 4.4_


- [ ] 9. Implement Inventory Service with Redis-based locking
  - [ ] 9.1 Create Inventory Lambda functions in Go
    - Implement checkAvailability to query Redis for real-time inventory (Go)
    - Implement lockInventory with Redis MULTI/EXEC atomic operations (Go)
    - Implement releaseLock to manually release inventory holds (Go)
    - Implement confirmBooking to move held spots to booked spots (Go)
    - Implement getInventoryStatus for admin dashboard (Go)
    - Implement updateCapacity for admin to adjust capacity (Go)
    - Implement setBlackoutDate to prevent bookings (Go)
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.10_
  
  - [ ] 9.2 Implement Redis inventory data structures
    - Create inventory:{excursion_id}:{date} hash with total_capacity, booked, held, available
    - Create hold:{booking_id} hash with TTL of 300 seconds
    - Implement automatic lock expiration with Redis TTL
    - Implement atomic decrement for available spots
    - _Requirements: 5.3, 5.4, 5.5, 5.8_
  
  - [ ] 9.3 Implement inventory consistency checks
    - Create Lambda in Go to sync Redis inventory with RDS persistent records
    - Implement CloudWatch alarm for inventory inconsistencies
    - Create reconciliation job to detect and fix discrepancies
    - _Requirements: 5.9, 19.9, 19.10_
  
  - [ ]* 9.4 Write property test for inventory consistency using gopter or rapid
    - **Property 1: Inventory Consistency - booked + held + available always equals total capacity**
    - **Validates: Requirements 5.2, 5.9, 19.1, 19.2, 19.9**
  
  - [ ]* 9.5 Write property test for no overbooking using gopter or rapid
    - **Property 2: No Overbooking - booked spots never exceed total capacity**
    - **Validates: Requirements 5.2, 19.2, 19.4, 19.5, 19.6**
  
  - [ ]* 9.6 Write unit tests for inventory service using Go testing
    - Test availability check with sufficient capacity
    - Test lock acquisition success and failure
    - Test concurrent lock attempts (race conditions)
    - Test lock expiration after TTL
    - Test inventory consistency maintenance
    - _Requirements: 5.1, 5.3, 5.4, 5.5, 5.8, 5.9_

- [ ] 10. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 11. Implement Pricing Engine with dynamic pricing
  - [ ] 11.1 Create Pricing Lambda functions in Go
    - Implement calculatePrice with base, tiered, and dynamic pricing logic (Go)
    - Implement getPricingTiers to retrieve pricing configuration (Go)
    - Implement setPricingRule for admin to configure pricing (Go)
    - Implement setSeasonalPricing for admin to set seasonal multipliers (Go)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9, 6.10_
  
  - [ ] 11.2 Implement dynamic pricing factors
    - Implement demand-based multiplier using capacity utilization thresholds
    - Implement seasonal multiplier based on date ranges
    - Implement cruise line premium pricing multiplier
    - Implement advance booking discount based on days in advance
    - Implement last-minute discount based on hours until departure
    - Calculate final price multiplier by combining all factors
    - _Requirements: 6.3, 6.4, 6.5, 6.6, 6.7_
  
  - [ ] 11.3 Implement price breakdown calculation
    - Calculate adult, child, and infant prices separately
    - Apply passenger type multipliers for tiered pricing
    - Generate detailed price breakdown with description, quantity, unit price, total
    - Sum all passenger types to calculate total price
    - _Requirements: 6.2, 6.8, 6.9_
  
  - [ ]* 11.4 Write property test for price positivity using gopter or rapid
    - **Property 5: Price Positivity - Total price and multipliers are always positive**
    - **Validates: Requirements 6.8, 6.10**
  
  - [ ]* 11.5 Write unit tests for pricing engine using Go testing
    - Test fixed pricing calculation
    - Test tiered pricing with adult/child/infant
    - Test dynamic pricing with demand multiplier
    - Test seasonal pricing adjustments
    - Test cruise line premium pricing
    - Test advance booking and last-minute discounts
    - Test price breakdown summation
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9_


- [ ] 12. Implement Payment Service with Stripe/Adyen integration
  - [ ] 12.1 Create Payment Lambda functions in Go
    - Implement createPaymentIntent with Stripe/Adyen SDK (Go)
    - Implement confirmPayment to verify payment status (Go)
    - Implement processRefund with refund amount and reason (Go)
    - Implement getPaymentStatus to query payment gateway (Go)
    - Store payment gateway credentials in Secrets Manager
    - _Requirements: 7.6, 7.7, 8.3, 8.4, 20.1, 20.2, 20.3, 20.7_
  
  - [ ] 12.2 Implement payment webhook handler
    - Create webhook Lambda in Go to handle payment gateway events
    - Verify webhook signature for security
    - Handle payment_intent.succeeded event
    - Handle payment_intent.failed event
    - Handle refund.completed event
    - Log all payment events for audit trail
    - _Requirements: 7.9, 7.10, 20.7, 30.3_
  
  - [ ] 12.3 Implement payment security measures
    - Use payment gateway client-side SDK for card tokenization
    - Never store credit card numbers in the system
    - Use TLS 1.2+ for all payment communications
    - Mask sensitive data in logs
    - Store payment intent IDs in PostgreSQL with encryption at rest
    - _Requirements: 20.1, 20.2, 20.3, 20.4, 20.5, 20.6, 20.8_
  
  - [ ]* 12.4 Write property test for payment idempotency using gopter or rapid
    - **Property 9: Payment Idempotency - Processing same payment intent multiple times results in single booking**
    - **Validates: Requirements 7.10, NFR-5**
  
  - [ ]* 12.5 Write unit tests for payment service using Go testing
    - Test payment intent creation
    - Test webhook signature verification
    - Test successful payment handling
    - Test failed payment handling
    - Test refund processing
    - _Requirements: 7.6, 7.7, 7.9, 7.10, 8.3, 8.4_


- [ ] 13. Implement Booking Service with complete lifecycle management
  - [ ] 13.1 Create Booking Lambda functions in Go
    - Implement createBooking with validation, inventory locking, and payment intent creation (Go)
    - Implement getBooking to retrieve booking details (Go)
    - Implement listUserBookings with filters and pagination (Go)
    - Implement cancelBooking with refund processing (Go)
    - Implement modifyBooking for date/passenger changes (Go)
    - Implement getVoucher to retrieve voucher PDF (Go)
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.8, 8.1, 8.2_
  
  - [ ] 13.2 Implement booking validation logic
    - Validate excursion is active and has available capacity
    - Validate cruise time window if cruisePortCallId provided
    - Validate flight earliest start if flightArrivalId provided
    - Validate booking cutoff (current time + cutoff hours <= departure time)
    - Validate passenger information completeness
    - Validate emergency contact phone number format (E.164)
    - _Requirements: 7.1, 7.2, 7.3, 15.1, 15.2, 18.2, 18.3, 18.4_
  
  - [ ] 13.3 Implement booking creation workflow
    - Check availability and validate eligibility
    - Lock inventory with 5-minute TTL
    - Calculate pricing with dynamic factors
    - Create payment intent
    - Create booking record with status PaymentHold
    - Return payment intent client secret to user
    - _Requirements: 7.4, 7.5, 7.6, 7.7, 7.8_
  
  - [ ] 13.4 Implement payment confirmation workflow
    - Handle payment webhook event
    - Update booking status to Confirmed
    - Update payment status to Paid
    - Move held inventory to booked inventory
    - Generate voucher PDF with QR code
    - Store voucher in S3 with signed URL
    - Publish BookingConfirmed event to EventBridge
    - Send booking confirmation to SQS queue
    - _Requirements: 7.10, 7.11, 7.12, 7.13, 7.14_
  
  - [ ]* 13.5 Write property test for booking atomicity using gopter or rapid
    - **Property 4: Booking Atomicity - Confirmed bookings have payment, inventory, and voucher**
    - **Validates: Requirements 7.10, 7.11, 7.12, 7.13**
  
  - [ ]* 13.6 Write property test for passenger count using gopter or rapid
    - **Property 6: Passenger Validation - Total passengers equals passenger array length**
    - **Validates: Requirements 7.1, 15.1**
  
  - [ ]* 13.7 Write unit tests for booking service using Go testing
    - Test booking creation with valid data
    - Test booking validation failures
    - Test time window validation
    - Test cutoff time enforcement
    - Test payment confirmation workflow
    - Test booking cancellation
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.10, 8.1, 18.3, 18.4_


- [ ] 14. Implement cancellation and refund processing
  - [ ] 14.1 Create cancellation Lambda functions in Go
    - Implement cancelBooking with cancellation policy calculation (Go)
    - Calculate cancellation fee based on excursion policy
    - Calculate refund amount (total price - cancellation fee)
    - Update booking status to Cancelled
    - Initiate refund with payment gateway
    - Release booked inventory back to available
    - Publish BookingCancelled event to EventBridge
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.6, 8.7, 8.8_
  
  - [ ] 14.2 Implement refund tracking
    - Create refund record with status Processing
    - Handle refund webhook events from payment gateway
    - Update refund status to Completed or Failed
    - Update booking payment status to Refunded or PartiallyRefunded
    - Send refund confirmation notification
    - _Requirements: 8.4, 8.5, 8.7, 8.9_
  
  - [ ]* 14.3 Write property test for refund constraints using gopter or rapid
    - **Property 12: Refund Constraint - Refund amount never exceeds original payment**
    - **Validates: Requirements 8.3, 8.4**
  
  - [ ]* 14.4 Write unit tests for cancellation service using Go testing
    - Test cancellation fee calculation
    - Test refund amount calculation
    - Test inventory release
    - Test refund processing
    - _Requirements: 8.1, 8.2, 8.3, 8.6_

- [ ] 15. Implement voucher generation with QR codes
  - [ ] 15.1 Create voucher generation Lambda in Go
    - Generate PDF voucher with booking details using Go PDF library (e.g., gofpdf)
    - Include excursion title, date, start time, meeting point, passenger names, booking ID
    - Generate QR code containing booking ID using Go QR library (e.g., go-qrcode)
    - Embed QR code image in PDF
    - Upload PDF to S3 with encryption
    - Generate signed URL with 1-year expiration
    - Update booking record with voucher URL
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5, 16.6_
  
  - [ ] 15.2 Implement QR code scanning for check-in
    - Create Lambda in Go to retrieve booking details from QR code
    - Update booking check-in status
    - Return passenger information for supplier verification
    - _Requirements: 16.8_
  
  - [ ]* 15.3 Write property test for voucher accessibility using gopter or rapid
    - **Property 15: Voucher Accessibility - Confirmed bookings have accessible vouchers with valid QR codes**
    - **Validates: Requirements 16.5, 16.6, 16.8**
  
  - [ ]* 15.4 Write unit tests for voucher generation using Go testing
    - Test PDF generation with booking data
    - Test QR code generation
    - Test S3 upload and signed URL generation
    - Test voucher retry on failure
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5, 16.9_

- [ ] 16. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 17. Implement event-driven notification system
  - [ ] 17.1 Set up EventBridge, SQS, and SNS infrastructure
    - Create EventBridge event bus for booking events
    - Create SQS queues: booking-confirmations, supplier-notifications, booking-reminders
    - Create SNS topics for email and SMS notifications
    - Configure dead-letter queues for failed messages
    - Set up SQS Lambda triggers with batch processing
    - _Requirements: 23.1, 23.2, 23.3, 23.4, 23.5, 23.6, 23.7, 23.10_
  
  - [ ] 17.2 Implement notification Lambda functions in Go
    - Create email notification Lambda using Amazon SES (Go)
    - Create SMS notification Lambda using Amazon Pinpoint (Go)
    - Implement booking confirmation email with voucher attachment
    - Implement supplier notification with passenger manifest
    - Implement booking reminder email (24 hours before departure)
    - Implement cancellation confirmation email
    - Implement refund confirmation email
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.9, 9.10_
  
  - [ ] 17.3 Implement event publishing
    - Publish BookingConfirmed event on successful payment
    - Publish BookingCancelled event on cancellation
    - Publish InventoryUpdated event on capacity changes
    - Publish PaymentCompleted event on payment success
    - Include all relevant context in events (booking ID, user ID, excursion ID, timestamp)
    - _Requirements: 23.1, 23.2, 23.3, 23.8, 23.9, 23.10_
  
  - [ ] 17.4 Implement retry logic with exponential backoff
    - Configure SQS message retry with exponential backoff
    - Set maximum retry attempts (3-5 retries)
    - Move failed messages to dead-letter queue
    - Create CloudWatch alarm for dead-letter queue depth
    - _Requirements: 23.6, 23.7_
  
  - [ ]* 17.5 Write unit tests for notification system using Go testing
    - Test email sending with SES
    - Test SMS sending with Pinpoint
    - Test event publishing to EventBridge
    - Test SQS message processing
    - Test retry logic and dead-letter queue
    - _Requirements: 9.1, 9.2, 9.8, 23.4, 23.5, 23.6_


- [ ] 18. Implement timezone handling across all services
  - [ ] 18.1 Create timezone utility functions in Go
    - Implement parseDateTime with IANA timezone support using Go time package
    - Implement convertToUTC for database storage
    - Implement convertToDestinationTimezone for display
    - Implement calculateHoursDifference in destination timezone
    - Implement validateTimezone to check IANA timezone validity
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8_
  
  - [ ] 18.2 Apply timezone handling to all date/time operations
    - Store all timestamps in UTC in database
    - Convert to destination timezone for time window calculations
    - Convert to destination timezone for cutoff validation
    - Parse cruise arrival/departure times with port timezone
    - Parse flight arrival times with airport timezone
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.7, 17.8_
  
  - [ ]* 18.3 Write property test for timezone consistency using gopter or rapid
    - **Property 10: Timezone Consistency - All timestamps stored in UTC and displayed in destination timezone**
    - **Validates: Requirements 17.1, 17.2**
  
  - [ ]* 18.4 Write unit tests for timezone handling using Go testing
    - Test UTC storage and retrieval
    - Test timezone conversion
    - Test time window calculations across timezones
    - Test cutoff validation with different timezones
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5_

- [ ] 19. Implement booking cutoff enforcement
  - [ ] 19.1 Create cutoff validation functions
    - Implement validateBookingCutoff to check hours until departure
    - Calculate hours difference in destination timezone
    - Compare with excursion schedule cutoff hours
    - Return validation error if within cutoff window
    - _Requirements: 18.1, 18.2, 18.3, 18.4_
  
  - [ ] 19.2 Apply cutoff filtering to search results
    - Filter out excursions where current time is within cutoff window
    - Display "Booking closed" message for excursions past cutoff
    - _Requirements: 18.5_
  
  - [ ]* 19.3 Write property test for booking cutoff using gopter or rapid
    - **Property 11: Booking Cutoff - All bookings respect minimum cutoff hours**
    - **Validates: Requirements 18.2, 18.3, 18.4**
  
  - [ ]* 19.4 Write unit tests for cutoff enforcement using Go testing
    - Test cutoff validation with various time differences
    - Test cutoff filtering in search results
    - Test cutoff updates by supplier
    - _Requirements: 18.2, 18.3, 18.4, 18.5, 18.6_


- [ ] 20. Implement Admin Service for suppliers and operations
  - [ ] 20.1 Create Supplier Dashboard Lambda functions in Go
    - Implement getSupplierDashboard with metrics (active excursions, bookings, revenue) (Go)
    - Implement manageExcursion for CRUD operations (Go)
    - Implement setBlackoutDates to prevent bookings (Go)
    - Implement getBookingManifest with passenger list and emergency contacts (Go)
    - Implement exportManifest to generate PDF manifest (Go)
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.9_
  
  - [ ] 20.2 Create Operations Dashboard Lambda functions in Go
    - Implement getOperationsDashboard with daily metrics (Go)
    - Implement getPassengerList for specific excursion and date (Go)
    - Implement trackCapacity with real-time utilization (Go)
    - Display upcoming departures within 24 hours
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.10_
  
  - [ ] 20.3 Create Reporting Lambda functions in Go
    - Implement getRevenueReport aggregated by destination, cruise line, category (Go)
    - Implement getCommissionReport for supplier payments (Go)
    - Implement getRefundReport with amounts and statuses (Go)
    - Implement exportReport to generate CSV/PDF files (Go)
    - Store reports in S3 with signed URLs
    - _Requirements: 11.5, 11.6, 11.7, 11.8_
  
  - [ ]* 20.4 Write unit tests for admin service using Go testing
    - Test supplier dashboard metrics calculation
    - Test manifest generation with passenger data
    - Test revenue report aggregation
    - Test commission calculation
    - _Requirements: 10.1, 10.5, 11.1, 11.5, 11.6_

- [ ] 21. Implement User Profile Management
  - [ ] 21.1 Create User Profile Lambda functions in Go
    - Implement getUserProfile to retrieve user data (Go)
    - Implement updateUserProfile with validation (Go)
    - Implement updateEmail with verification workflow (Go)
    - Implement getBookingHistory with filters (Go)
    - Implement updateNotificationPreferences (Go)
    - Implement disconnectSocialAccount (Go)
    - Implement deleteAccount with GDPR compliance (Go)
    - Implement exportUserData for GDPR data portability (Go)
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7, 14.8, 14.9, 14.10_
  
  - [ ]* 21.2 Write unit tests for user profile service using Go testing
    - Test profile updates
    - Test email verification
    - Test booking history retrieval
    - Test account deletion
    - Test data export
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.9, 14.10_


- [ ] 22. Implement Review and Rating System
  - [ ] 22.1 Create Review Lambda functions in Go
    - Implement submitReview with validation (completed booking required) (Go)
    - Implement getReviews for excursion with pagination (Go)
    - Implement voteHelpful to increment helpful count (Go)
    - Implement addSupplierResponse for supplier replies (Go)
    - Validate user can only review once per booking
    - Mark reviews as verified purchase
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_
  
  - [ ] 22.2 Implement rating calculation
    - Recalculate excursion average rating on new review
    - Update review count
    - Store ratings in excursion record for fast retrieval
    - _Requirements: 13.1, 13.9_
  
  - [ ]* 22.3 Write unit tests for review system using Go testing
    - Test review submission validation
    - Test one review per booking constraint
    - Test rating calculation
    - Test helpful votes
    - _Requirements: 13.3, 13.4, 13.5, 13.8, 13.9_

- [ ] 23. Implement AI-powered Recommendation Engine
  - [ ] 23.1 Create Recommendation Lambda functions in Go
    - Implement getRecommendations with user booking history (Go)
    - Implement getCrossSellRecommendations for complementary excursions (Go)
    - Implement getUpsellRecommendations for premium alternatives (Go)
    - Implement getTrendingExcursions by destination (Go)
    - Filter recommendations by time window if cruise/flight context exists
    - Filter recommendations by availability
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.7, 12.8_
  
  - [ ] 23.2 Implement collaborative filtering
    - Query user booking history from RDS
    - Find similar users based on booking patterns
    - Recommend excursions booked by similar users
    - _Requirements: 12.1, 12.5_
  
  - [ ] 23.3 Implement content-based filtering
    - Analyze excursion categories, difficulty, price range
    - Find similar excursions based on attributes
    - Recommend based on user preferences
    - _Requirements: 12.6_
  
  - [ ]* 23.4 Write unit tests for recommendation engine using Go testing
    - Test personalized recommendations
    - Test cross-sell logic
    - Test upsell logic
    - Test time window filtering
    - _Requirements: 12.1, 12.2, 12.3, 12.7_


- [ ] 24. Implement weather cancellation and force majeure handling
  - [ ] 24.1 Create supplier cancellation Lambda functions in Go
    - Implement cancelExcursionBySupplier with reason (weather, force majeure, operational) (Go)
    - Update all affected bookings to Cancelled status
    - Initiate full refunds for all affected bookings (waive cancellation fees)
    - Release booked inventory back to available
    - Send notification emails to all affected users
    - Prioritize notifications for cancellations within 24 hours
    - _Requirements: 29.1, 29.2, 29.3, 29.4, 29.5, 29.6, 29.7_
  
  - [ ]* 24.2 Write unit tests for supplier cancellation using Go testing
    - Test bulk booking cancellation
    - Test full refund processing
    - Test inventory release
    - Test notification sending
    - _Requirements: 29.1, 29.2, 29.3, 29.4_

- [ ] 25. Implement audit trail and compliance logging
  - [ ] 25.1 Create audit logging functions
    - Log booking creation with user ID, timestamp, excursion ID, details
    - Log booking status changes with old/new status, timestamp, actor
    - Log payment processing with payment intent ID, amount, currency, status
    - Log refund processing with refund ID, amount, reason
    - Log inventory updates with old/new capacity, timestamp, actor
    - Log personal data access with accessor, timestamp, data accessed
    - Log personal data deletion with timestamp (retain audit record)
    - _Requirements: 30.1, 30.2, 30.3, 30.4, 30.5, 30.6, 30.7_
  
  - [ ] 25.2 Set up CloudTrail for AWS API logging
    - Enable CloudTrail for all AWS API calls
    - Configure log retention for 7 years
    - Set up S3 bucket for long-term audit log storage
    - _Requirements: 30.8, 30.9_
  
  - [ ] 25.3 Implement audit log query interface
    - Create Lambda in Go to search audit logs by user, action, timestamp, resource
    - Implement filtering and pagination
    - _Requirements: 30.10_
  
  - [ ]* 25.4 Write unit tests for audit logging using Go testing
    - Test booking audit trail
    - Test payment audit trail
    - Test data access logging
    - _Requirements: 30.1, 30.2, 30.3, 30.6_

- [ ] 26. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 27. Implement monitoring, observability, and alerting
  - [ ] 27.1 Set up CloudWatch metrics and dashboards
    - Create custom metrics for booking conversion rate, inventory utilization, payment success rate
    - Create CloudWatch dashboard with key metrics (Lambda duration, API latency, RDS connections, Redis hit rate)
    - Configure metric filters for error patterns
    - _Requirements: 24.1, 24.2, 24.3, 24.4, 24.10_
  
  - [ ] 27.2 Configure CloudWatch alarms
    - Create alarm for error rate > 1%
    - Create alarm for API latency exceeding SLA thresholds
    - Create alarm for inventory inconsistencies
    - Create alarm for payment success rate < 95%
    - Create alarm for dead-letter queue depth
    - Configure SNS topic for alarm notifications
    - Integrate with PagerDuty for on-call alerting
    - _Requirements: 24.5, 24.6, 24.7, 24.8, 27.10_
  
  - [ ] 27.3 Set up X-Ray distributed tracing
    - Enable X-Ray tracing for API Gateway
    - Enable X-Ray tracing for Go Lambda functions
    - Instrument critical code paths with X-Ray segments in Go
    - Create X-Ray service map for end-to-end request flow
    - _Requirements: 24.9_
  
  - [ ] 27.4 Configure structured logging
    - Implement structured JSON logging for all Go Lambda functions
    - Include correlation IDs for request tracing
    - Log errors with stack traces
    - Configure log retention (30 days for application logs)
    - _Requirements: 24.1, NFR-7_
  
  - [ ]* 27.5 Write unit tests for monitoring using Go testing
    - Test custom metric publishing
    - Test alarm triggering conditions
    - Test structured log formatting
    - _Requirements: 24.1, 24.10_


- [ ] 28. Implement API security and rate limiting
  - [ ] 28.1 Configure API Gateway security
    - Set up Cognito authorizer for protected endpoints
    - Implement JWT token validation on every request
    - Configure CORS with appropriate origins
    - Set up request throttling (10,000 requests/sec per account)
    - Configure burst limits (5,000 requests)
    - _Requirements: 22.1, 22.2, 22.9_
  
  - [ ]* 28.2 Configure AWS WAF rules (optional - future backlog)
    - Create WAF web ACL for API Gateway
    - Add SQL injection protection rule
    - Add XSS protection rule
    - Add rate limiting rule (1,000 requests per 5 minutes per IP)
    - Add geo-blocking rules if needed
    - _Requirements: 22.4, 22.5, 22.6_
  
  - [ ] 28.3 Implement input validation and sanitization
    - Create JSON schema validators for all API endpoints
    - Implement input sanitization to prevent XSS
    - Use parameterized queries for all database operations
    - Validate all user inputs before processing
    - _Requirements: 22.7, 22.8_
  
  - [ ] 28.4 Implement role-based access control
    - Define IAM roles for User, Supplier, Admin
    - Implement resource-level permissions (users access only their bookings)
    - Return HTTP 403 Forbidden for unauthorized access
    - _Requirements: 22.3, NFR-4_
  
  - [ ]* 28.5 Write unit tests for API security using Go testing
    - Test JWT token validation
    - Test unauthorized access rejection
    - Test input validation
    - Test SQL injection prevention
    - _Requirements: 22.1, 22.2, 22.3, 22.7, 22.8_

- [ ] 29. Implement data encryption and privacy controls
  - [ ] 29.1 Configure encryption at rest
    - Enable AES-256 encryption for Aurora Serverless v2 with KMS
    - Enable encryption for ElastiCache Redis
    - Enable server-side encryption for S3 buckets (SSE-KMS)
    - _Requirements: 21.1, 21.2, 21.3_
  
  - [ ] 29.2 Configure encryption in transit
    - Enforce TLS 1.2+ for all API communications
    - Configure HTTPS-only for CloudFront (redirect HTTP to HTTPS)
    - Use VPC endpoints for AWS service communication
    - _Requirements: 21.5, NFR-4_
  
  - [ ] 29.3 Implement application-level encryption for sensitive data
    - Encrypt passport numbers before storing in database
    - Encrypt bank account details for suppliers
    - Use AWS KMS for encryption key management
    - _Requirements: 21.6, 21.7_
  
  - [ ] 29.4 Implement data masking in logs
    - Mask email addresses in application logs
    - Mask phone numbers in application logs
    - Never log credit card data
    - _Requirements: 21.8, 20.6_
  
  - [ ] 29.5 Implement GDPR compliance features
    - Implement account deletion with permanent data erasure
    - Implement data export in JSON format
    - Ensure encrypted fields are deleted on account deletion
    - _Requirements: 21.9, 21.10, 14.9, 14.10_
  
  - [ ]* 29.6 Write unit tests for encryption and privacy using Go testing
    - Test data encryption/decryption
    - Test log masking
    - Test account deletion completeness
    - _Requirements: 21.6, 21.7, 21.8, 21.9_


- [ ] 30. Implement performance optimizations
  - [ ] 30.1 Configure caching layers
    - Set up ElastiCache Redis for inventory (sub-millisecond reads)
    - Configure CloudFront CDN for static assets with cache headers
    - Set up API Gateway caching for destination/excursion lists (TTL: 5 minutes)
    - Use PostgreSQL with JSONB for session data with indexed queries
    - Configure pg_cron for automated session cleanup based on expires_at
    - _Requirements: 26.1, 26.2, 26.8, NFR-1_
  
  - [ ] 30.2 Optimize database queries
    - Create indexes on destination_id, date, cruise_line, excursion_id
    - Configure Aurora Serverless v2 read replicas for search queries (if needed)
    - Implement connection pooling with RDS Proxy
    - Partition large tables (bookings by date)
    - _Requirements: 26.7, NFR-1_
  
  - [ ] 30.3 Configure Lambda optimizations
    - Set up provisioned concurrency for critical Go Lambda functions (booking, payment)
    - Tune Lambda memory allocation (1024MB-3008MB based on profiling)
    - Implement code bundling for Go binaries
    - Use Lambda layers for shared Go dependencies
    - _Requirements: 26.6, NFR-2_
  
  - [ ] 30.4 Implement asynchronous processing
    - Use SQS for non-critical operations (email, reporting) with Go Lambda consumers
    - Use EventBridge for event-driven workflows
    - Use Lambda async invocation for voucher generation
    - Implement batch processing for supplier manifests in Go
    - _Requirements: 26.10, NFR-2_
  
  - [ ]* 30.5 Write performance tests using Go testing
    - Test search latency under load
    - Test availability check latency
    - Test booking creation latency
    - Test concurrent booking scenarios
    - _Requirements: 26.1, 26.2, 26.3, 26.5_

- [ ] 31. Implement disaster recovery and high availability
  - [ ] 31.1 Configure backup and recovery
    - Enable Aurora Serverless v2 automated daily backups with 7-day retention
    - Enable Aurora Serverless v2 point-in-time recovery
    - Enable S3 versioning for vouchers
    - Configure cross-region replication for S3
    - _Requirements: 27.3, 27.4, 27.5_
  
  - [ ] 31.2 Configure high availability
    - Deploy Lambda functions across multiple availability zones
    - Configure Aurora Serverless v2 with automatic failover within region
    - Configure ElastiCache Redis cluster mode with automatic failover
    - Set up multi-region active-passive deployment
    - _Requirements: 27.1, 27.6, 27.7, 27.8, 27.9, NFR-3_
  
  - [ ] 31.3 Set up disaster recovery procedures
    - Document RTO (1 hour) and RPO (5 minutes) targets
    - Create runbooks for failover procedures
    - Configure SNS notifications for critical alarms
    - Integrate with PagerDuty for on-call team
    - _Requirements: 27.1, 27.2, 27.10_
  
  - [ ]* 31.4 Write disaster recovery tests using Go testing
    - Test database backup and restore
    - Test failover procedures
    - Test cross-region replication
    - _Requirements: 27.1, 27.2, 27.3, 27.4_


- [ ] 32. Implement external API integrations
  - [ ] 32.1 Integrate cruise schedule APIs
    - Implement Lambda in Go to fetch cruise schedules from external providers
    - Parse cruise schedule data and store in RDS
    - Implement caching strategy (24 hours)
    - Handle API failures with cached data fallback
    - Implement retry logic with exponential backoff
    - Set API timeout to 10 seconds
    - _Requirements: 3.7, 7.1_
  
  - [ ] 32.2 Integrate flight schedule APIs
    - Implement Lambda in Go to fetch flight data from FlightAware/Amadeus
    - Parse flight arrival data and store in RDS
    - Implement caching strategy (1 hour)
    - Handle API failures gracefully
    - Update flight status on delays/cancellations
    - _Requirements: 4.5, 4.6_
  
  - [ ] 32.3 Implement circuit breaker pattern
    - Create circuit breaker for external API calls
    - Open circuit after consecutive failures
    - Implement half-open state for recovery testing
    - Gracefully degrade when APIs unavailable
    - _Requirements: NFR-5_
  
  - [ ]* 32.4 Write integration tests for external APIs using Go testing
    - Test cruise schedule ingestion
    - Test flight schedule ingestion
    - Test caching behavior
    - Test circuit breaker functionality
    - _Requirements: 3.7, 4.5_

- [ ] 33. Implement accessibility and internationalization
  - [ ] 33.1 Implement multi-language support
    - Create language resource files (English, Spanish, French, German)
    - Implement language selection in user preferences (backend Go)
    - Display all UI text in selected language (frontend React/TypeScript)
    - _Requirements: 28.1_
  
  - [ ] 33.2 Implement locale-aware formatting
    - Format currency according to user locale (backend Go)
    - Format dates according to user locale (backend Go)
    - _Requirements: 28.2, 28.3_
  
  - [ ] 33.3 Implement accessibility features
    - Display wheelchair accessibility prominently (frontend React/TypeScript)
    - Allow filtering by accessibility features (backend Go + frontend React/TypeScript)
    - Ensure proper ARIA labels for screen readers (frontend React/TypeScript)
    - Maintain WCAG 2.1 AA color contrast ratios (frontend React/TypeScript)
    - Support keyboard navigation (frontend React/TypeScript)
    - _Requirements: 28.4, 28.5, 28.6, 28.7, 28.8_
  
  - [ ]* 33.4 Write unit tests for internationalization using Go testing
    - Test language selection
    - Test currency formatting
    - Test date formatting
    - _Requirements: 28.1, 28.2, 28.3_


- [ ] 34. Implement emergency contact and special requests handling
  - [ ] 34.1 Create emergency contact validation
    - Validate emergency contact name, phone, relationship are required
    - Validate phone number is in E.164 format
    - Encrypt emergency contact phone numbers at rest
    - _Requirements: 15.1, 15.2, 15.6_
  
  - [ ] 34.2 Implement special requests handling
    - Allow text field for special requests during booking
    - Display special requests in supplier manifest
    - Prominently display dietary restrictions and medical conditions
    - _Requirements: 15.3, 15.4, 15.5, 15.7_
  
  - [ ]* 34.3 Write unit tests for emergency contact handling using Go testing
    - Test phone number validation
    - Test E.164 format enforcement
    - Test encryption of phone numbers
    - _Requirements: 15.1, 15.2, 15.6_

- [ ] 35. Set up CI/CD pipeline and deployment
  - [ ] 35.1 Configure CI/CD pipeline
    - Set up GitHub/GitLab repository
    - Configure AWS CodeBuild for build stage
    - Configure automated test execution (unit, integration, property-based)
    - Set up AWS CodePipeline for deployment
    - Configure AWS CodeDeploy for Lambda deployments
    - _Requirements: NFR-6_
  
  - [ ] 35.2 Implement blue-green deployment
    - Configure Lambda aliases for traffic shifting
    - Set up API Gateway stages (blue/green)
    - Implement gradual rollout (10% → 50% → 100%)
    - Configure automatic rollback on CloudWatch alarms
    - _Requirements: NFR-6_
  
  - [ ] 35.3 Create deployment environments
    - Set up development environment with LocalStack
    - Set up staging environment in AWS
    - Set up production environment with multi-region support
    - Configure environment-specific variables
    - _Requirements: NFR-6_
  
  - [ ]* 35.4 Write deployment tests using Go testing
    - Test infrastructure provisioning
    - Test Lambda deployment
    - Test API Gateway configuration
    - Test rollback procedures
    - _Requirements: NFR-6_

- [ ] 36. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 37. Integration testing and end-to-end validation
  - [ ]* 37.1 Write integration test for complete booking flow
    - Test user authentication → excursion search → booking creation → payment → confirmation
    - Verify inventory locked → payment processed → inventory updated → voucher generated → emails sent
    - _Requirements: 1.4, 2.3, 7.1, 7.10, 7.11, 7.12, 7.13, 7.14_
  
  - [ ]* 37.2 Write integration test for cruise schedule integration
    - Test cruise schedule ingestion → time window calculation → excursion filtering
    - Verify timezone handling and eligibility filtering
    - _Requirements: 3.1, 3.2, 3.3, 3.8_
  
  - [ ]* 37.3 Write integration test for payment gateway
    - Test payment intent creation → successful payment → webhook handling → booking confirmation
    - Test payment failure scenarios
    - _Requirements: 7.6, 7.7, 7.9, 7.10_
  
  - [ ]* 37.4 Write integration test for event-driven workflows
    - Test booking confirmed → EventBridge → SNS → SES (email sent)
    - Test booking confirmed → SQS → Lambda → supplier notification
    - _Requirements: 9.1, 9.2, 23.1, 23.2_
  
  - [ ]* 37.5 Write integration test for Redis inventory locking
    - Test concurrent booking attempts
    - Test lock expiration
    - Test inventory consistency under load
    - _Requirements: 5.3, 5.4, 5.5, 5.8, 19.6_

- [ ] 38. Documentation and API specification
  - Create OpenAPI specification for all API endpoints
  - Document authentication flows
  - Document error codes and responses
  - Create developer guide for API integration
  - Document deployment procedures
  - Create operations runbook
  - _Requirements: NFR-6_

- [ ] 39. Final system validation and deployment
  - Run full test suite (unit, integration, property-based)
  - Perform load testing to validate performance targets
  - Validate security controls (WAF, encryption, authentication)
  - Deploy to staging environment
  - Perform user acceptance testing
  - Deploy to production with gradual rollout
  - Monitor CloudWatch metrics and alarms
  - _Requirements: NFR-1, NFR-2, NFR-3, NFR-4, NFR-5_

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using gopter or rapid
- Unit tests validate specific examples and edge cases using Go testing
- Integration tests validate end-to-end workflows
- Backend Lambda functions are written in Go (Golang)
- Frontend UI components are written in React with TypeScript
- Infrastructure code uses AWS CDK with TypeScript
- Follow AWS best practices for serverless architecture
- Ensure all sensitive data is encrypted at rest and in transit
- Implement comprehensive error handling and logging
- Use infrastructure as code (AWS CDK) for all AWS resources



- [ ] 40. Implement Itinerary Management Service
  - [ ] 40.1 Create Itinerary Lambda functions in Go
    - Implement createItinerary with user ID, destination, dates (Go)
    - Implement getItinerary to retrieve itinerary details (Go)
    - Implement listUserItineraries with pagination (Go)
    - Implement updateItinerary for name, dates, notes (Go)
    - Implement deleteItinerary with cascade deletion of items (Go)
    - Implement shareItinerary with access control (view/edit permissions) (Go)
    - Implement getSharedItineraries for collaborative planning (Go)
    - _Requirements: 31.1, 31.2, 31.3, 31.4, 31.5, 32.1, 32.2_
  
  - [ ] 40.2 Create Itinerary Item Lambda functions in Go
    - Implement addItineraryItem (excursion, restaurant, shopping, custom) (Go)
    - Implement updateItineraryItem for time, notes, status (Go)
    - Implement removeItineraryItem (Go)
    - Implement reorderItineraryItems for drag-and-drop (Go)
    - Store item type (excursion, dining, shopping, custom) and reference ID
    - _Requirements: 31.6, 31.7, 31.8_
  
  - [ ] 40.3 Implement conflict detection
    - Implement detectConflicts to check time overlaps (Go)
    - Calculate item duration based on type (excursion duration, dining 2hrs, shopping 1hr)
    - Check if items overlap in time
    - Return conflict warnings with affected items
    - Implement autoResolveConflicts with time adjustment suggestions (Go)
    - _Requirements: 33.1, 33.2, 33.3, 33.4_
  
  - [ ] 40.4 Implement itinerary sharing and collaboration
    - Create share tokens with expiration dates
    - Implement access control (view-only vs edit permissions)
    - Track share recipients and access history
    - Send share invitation emails with link
    - Implement real-time collaboration with optimistic locking
    - _Requirements: 32.1, 32.2, 32.3, 32.4, 32.5_
  
  - [ ]* 40.5 Write property test for itinerary consistency using gopter or rapid
    - **Property 16: Itinerary Consistency - All itinerary items belong to valid itinerary**
    - **Validates: Requirements 31.1, 31.6**
  
  - [ ]* 40.6 Write unit tests for itinerary service using Go testing
    - Test itinerary creation and retrieval
    - Test item addition and reordering
    - Test conflict detection with overlapping times
    - Test sharing with permissions
    - Test collaborative editing
    - _Requirements: 31.1, 31.6, 32.1, 33.1_


- [ ] 41. Implement Dining & Shopping Service
  - [ ] 41.1 Create Dining Lambda functions in Go
    - Implement searchRestaurants with filters (cuisine, price, location) (Go)
    - Implement getRestaurant with details and reviews (Go)
    - Implement syncGooglePlaces to import restaurant data (Go)
    - Implement addToItinerary to add restaurant to trip plan (Go)
    - Store Google Place ID, rating, price level, opening hours
    - Generate affiliate links for reservation platforms (OpenTable, Resy)
    - Track click-through and conversion for commission calculation
    - _Requirements: 34.1, 34.2, 34.3, 34.4, 34.5_
  
  - [ ] 41.2 Create Shopping Lambda functions in Go
    - Implement searchShoppingLocations with filters (category, location) (Go)
    - Implement getShoppingLocation with details (Go)
    - Implement listLocalProducts with pricing and descriptions (Go)
    - Implement syncGooglePlaces for shopping venues (Go)
    - Generate affiliate links for e-commerce platforms
    - Track affiliate conversions for revenue reporting
    - _Requirements: 35.1, 35.2, 35.3, 35.4_
  
  - [ ] 41.3 Integrate Google Places API
    - Implement Lambda in Go to fetch place details from Google Places API
    - Parse place data (name, address, rating, photos, hours, phone)
    - Store place data in RDS with last_synced timestamp
    - Implement incremental sync (update changed places only)
    - Handle API rate limits with exponential backoff
    - Cache place data for 24 hours
    - _Requirements: 36.1, 36.2, 36.3, 36.4_
  
  - [ ] 41.4 Implement affiliate link generation
    - Create affiliate link templates for each partner platform
    - Generate tracking parameters (user_id, destination_id, timestamp)
    - Store affiliate clicks in RDS for analytics
    - Implement conversion tracking webhook handlers
    - Calculate commission revenue (5-15% for dining, 3-10% for shopping)
    - _Requirements: 42.1, 42.2, 42.3_
  
  - [ ]* 41.5 Write unit tests for dining and shopping service using Go testing
    - Test restaurant search with filters
    - Test Google Places sync
    - Test affiliate link generation
    - Test conversion tracking
    - _Requirements: 34.1, 35.1, 36.1, 42.1_


- [ ] 42. Implement Places Service for Google Places integration
  - [ ] 42.1 Create Places Lambda functions in Go
    - Implement syncPlaces to fetch and store place data (Go)
    - Implement searchPlaces with text search and filters (Go)
    - Implement getPlaceDetails with photos and reviews (Go)
    - Implement getNearbyPlaces with radius search (Go)
    - Store place types (restaurant, shopping, attraction, etc.)
    - _Requirements: 36.1, 36.2, 36.3_
  
  - [ ] 42.2 Implement place data enrichment
    - Fetch place photos from Google Places API
    - Store photo references and generate signed URLs
    - Parse and store opening hours with timezone
    - Extract and store place attributes (wheelchair accessible, outdoor seating)
    - _Requirements: 36.4_
  
  - [ ] 42.3 Implement place search optimization
    - Index places in Elasticsearch for fast text search
    - Implement autocomplete with edge n-grams
    - Implement geospatial search with distance calculation
    - Cache popular searches for 1 hour
    - _Requirements: 36.1, 36.2_
  
  - [ ]* 42.4 Write unit tests for places service using Go testing
    - Test place sync from Google API
    - Test place search with filters
    - Test nearby search with radius
    - Test photo URL generation
    - _Requirements: 36.1, 36.2, 36.3_


- [ ] 43. Implement Search Integration Service for trending data
  - [ ] 43.1 Create Search Integration Lambda functions in Go
    - Implement fetchTrendingSearches from Google Trends API (Go)
    - Implement analyzeTrends to identify popular destinations/activities (Go)
    - Implement getTrendingExcursions by destination (Go)
    - Implement getTrendingRestaurants by destination (Go)
    - Store trend data with timestamp and popularity score
    - _Requirements: 39.1, 39.2, 39.3_
  
  - [ ] 43.2 Implement trend analysis
    - Calculate trend score based on search volume change
    - Identify rising trends (>50% increase in 7 days)
    - Identify declining trends (<-30% decrease in 7 days)
    - Weight trends by geographic proximity to destinations
    - Update trend scores daily via scheduled Lambda
    - _Requirements: 39.1, 39.2_
  
  - [ ] 43.3 Integrate trending data into recommendations
    - Boost recommendation scores for trending items (+25% popularity weight)
    - Display "Trending" badge on popular excursions/restaurants
    - Create trending section on homepage
    - _Requirements: 39.3_
  
  - [ ]* 43.4 Write unit tests for search integration using Go testing
    - Test trend data fetching
    - Test trend score calculation
    - Test trending item identification
    - Test recommendation score boosting
    - _Requirements: 39.1, 39.2, 39.3_


- [ ] 44. Implement Enhanced Recommendation Engine with multi-factor scoring
  - [ ] 44.1 Create Enhanced Recommendation Lambda functions in Go
    - Implement getPersonalizedRecommendations with multi-factor scoring (Go)
    - Implement calculateRecommendationScore with weighted factors (Go)
    - Implement getUserSegment based on behavior and preferences (Go)
    - Implement getSegmentRecommendations for targeted suggestions (Go)
    - Store recommendation scores in Elasticsearch for fast retrieval
    - _Requirements: 37.1, 37.2, 37.3, 38.1, 38.2_
  
  - [ ] 44.2 Implement context-based scoring (30% weight)
    - Score based on cruise/flight time windows
    - Score based on destination proximity
    - Score based on weather conditions
    - Score based on current date/season
    - Score based on group size and composition
    - _Requirements: 37.1, 37.2_
  
  - [ ] 44.3 Implement behavior-based scoring (25% weight)
    - Analyze user booking history (categories, price range, difficulty)
    - Track user browsing behavior (views, searches, clicks)
    - Identify user preferences from past interactions
    - Calculate similarity to other users for collaborative filtering
    - _Requirements: 37.2, 37.3_
  
  - [ ] 44.4 Implement popularity-based scoring (25% weight)
    - Calculate booking frequency by destination
    - Weight by review ratings and count
    - Boost trending items from Google Trends data
    - Consider seasonal popularity patterns
    - _Requirements: 37.2, 39.3_
  
  - [ ] 44.5 Implement personalization scoring (20% weight)
    - Match excursion attributes to user segment
    - Apply segment-specific preferences (luxury, family, budget, food, adventure)
    - Boost items matching user's stated interests
    - Consider user's price sensitivity
    - _Requirements: 37.3, 38.1, 38.2_
  
  - [ ] 44.6 Implement user segmentation
    - Define 5 user segments: Luxury Travelers, Family Travelers, Budget Travelers, Food-Focused Travelers, Adventure Seekers
    - Implement segment classification based on booking patterns
    - Store user segment in user profile
    - Update segment periodically based on new bookings
    - Create segment-specific recommendation strategies
    - _Requirements: 38.1, 38.2, 38.3, 38.4, 38.5_
  
  - [ ] 44.7 Implement cross-selling and upselling
    - Identify complementary items (excursion + dining + shopping)
    - Suggest premium alternatives with higher margins
    - Bundle recommendations for multi-day itineraries
    - Calculate bundle discounts for increased conversion
    - Track cross-sell and upsell conversion rates
    - _Requirements: 42.1, 42.2, 42.3_
  
  - [ ]* 44.8 Write property test for recommendation scoring using gopter or rapid
    - **Property 17: Recommendation Score Validity - All scores are between 0 and 1, weights sum to 100%**
    - **Validates: Requirements 37.1, 37.2**
  
  - [ ]* 44.9 Write unit tests for enhanced recommendation engine using Go testing
    - Test multi-factor score calculation
    - Test user segmentation classification
    - Test context-based scoring
    - Test behavior-based scoring
    - Test cross-sell recommendations
    - _Requirements: 37.1, 37.2, 37.3, 38.1, 42.1_


- [ ] 45. Implement AgentCore Service (Decoupled AI Service)
  - [ ] 45.1 Set up AgentCore Gateway API
    - Create dedicated API Gateway for AgentCore service
    - Configure JWT authorizer using main Cognito user pool
    - Set up rate limiting (10 requests/minute per user)
    - Configure CORS for frontend access
    - Create endpoints: /conversations, /conversations/{id}/messages, /itineraries/generate, /health
    - Enable CloudWatch logging for all requests
    - _Requirements: 40.1, 40.2_
  
  - [ ] 45.2 Configure AWS Bedrock Guardrails for PII/PCI compliance
    - Create Bedrock Guardrail with sensitive information filters
    - Configure PII filters: BLOCK credit cards, MASK emails/phones/names/addresses
    - Configure content filters: BLOCK hate speech, insults, sexual content, violence (high confidence)
    - Configure denied topics: financial/medical/legal advice, political opinions
    - Enable prompt attack detection (jailbreak, injection)
    - Set blocked input/output messaging
    - Test guardrail with sample inputs containing PII
    - _Requirements: 40.5, 22.7, 22.8, 21.6, 21.7, 21.8_
  
  - [ ] 45.3 Implement Prompt Management with AWS Bedrock
    - Create versioned prompt templates in AWS Bedrock Prompt Management
    - Create trip-planner-v1 prompt for general trip planning
    - Create cruise-planner-v1 prompt for cruise-specific planning
    - Create flight-planner-v1 prompt for flight arrival planning
    - Create excursion-recommender-v1 prompt for excursion suggestions
    - Create dining-recommender-v1 prompt for restaurant suggestions
    - Create shopping-recommender-v1 prompt for shopping suggestions
    - Create conflict-detector-v1 and conflict-resolver-v1 prompts
    - Store prompt versions in S3 for backup
    - _Requirements: 40.1, 40.2_
  
  - [ ] 45.4 Implement LLM Gateway API Lambda in Go
    - Create Lambda function for LLM abstraction layer (Go)
    - Implement model selection logic (Claude 3 Sonnet primary, Haiku fallback)
    - Implement request formatting with prompt templates
    - Implement response parsing and error handling
    - Enable streaming support for real-time responses
    - Implement prompt caching for system prompts (50% cost reduction)
    - Implement context window management (limit to 100K tokens)
    - Track token usage and costs per request
    - Implement retry logic with exponential backoff
    - _Requirements: 40.1, 40.2, NFR-1, NFR-2_
  
  - [ ] 45.5 Implement AgentCore Orchestrator Lambda in Go
    - Create main conversation handler Lambda (Go)
    - Receive user messages from AgentCore Gateway API
    - Load conversation context from Memory Manager
    - Apply input guardrails to user messages
    - Call LLM Gateway API with prompt template and context
    - Apply output guardrails to LLM responses
    - Update conversation state via Memory Manager
    - Save messages to memory
    - Return filtered response to user
    - Handle MCP tool calls from LLM
    - _Requirements: 40.1, 40.2, 40.3, 40.4_
  
  - [ ] 45.6 Implement AgentCore Memory Manager Lambda in Go
    - Create memory management Lambda (Go)
    - Implement loadShortTermMemory (last 50 messages from Aurora)
    - Implement loadLongTermMemory (preferences, facts, summaries from Aurora)
    - Implement saveMessage to agentcore_messages table
    - Implement updateConversationState to agentcore_conversation_state table
    - Implement extractPreferences from conversation messages
    - Implement extractSemanticFacts from conversation messages
    - Implement generateConversationSummary using LLM
    - Implement saveToLongTermMemory (preferences, facts, summaries)
    - Implement memory retrieval with relevance scoring
    - _Requirements: 41.1, 41.2, 41.3_
  
  - [ ] 45.7 Create AgentCore database schema (Aurora PostgreSQL)
    - Create agentcore_conversations table (conversation_id, user_id, actor_id, session_id, expires_at, status)
    - Create agentcore_messages table (message_id, conversation_id, role, content, tokens_used, timestamp)
    - Create agentcore_conversation_state table (state_id, conversation_id, state_key, state_value JSONB)
    - Create agentcore_user_preferences table (preference_id, user_id, preference_type, preference_value, confidence_score)
    - Create agentcore_semantic_memory table (memory_id, user_id, fact_type, fact_content, confidence_score)
    - Create agentcore_conversation_summaries table (summary_id, user_id, session_id, summary_text, key_decisions JSONB)
    - Create agentcore_successful_itineraries table (itinerary_id, user_id, destination_id, user_segment, itinerary_data JSONB, s3_snapshot_key)
    - Create indexes on user_id, conversation_id, expires_at, preference_type, fact_type
    - Configure pg_cron for TTL cleanup (delete conversations older than 7 days)
    - _Requirements: 40.1, 41.1, 41.2_
  
  - [ ] 45.8 Implement MCP Client Executor Lambda in Go
    - Create MCP tool executor Lambda (Go)
    - Implement MCP tool registry (search_excursions, get_excursion_details, check_availability, create_booking)
    - Implement dining tools (search_restaurants, get_restaurant_details, get_dining_recommendations)
    - Implement shopping tools (search_shopping_locations, get_local_products)
    - Implement itinerary tools (create_itinerary, add_itinerary_item, detect_conflicts, resolve_conflicts)
    - Implement user tools (get_user_preferences, get_user_bookings, get_user_segment)
    - Apply input guardrails to all tool parameters
    - Call main application APIs for tool execution
    - Apply output guardrails to all tool responses
    - Format tool responses for LLM consumption
    - Handle tool errors gracefully with fallback messages
    - _Requirements: 40.1, 40.2, 40.3, 31.1, 34.1, 35.1_
  
  - [ ] 45.9 Implement Itinerary Generator Lambda in Go
    - Create AI-powered itinerary generator Lambda (Go)
    - Parse user intent (destination, dates, interests, budget, group composition)
    - Query available excursions via MCP tools
    - Query restaurants via MCP tools
    - Query shopping locations via MCP tools
    - Generate optimized day-by-day itinerary
    - Detect scheduling conflicts automatically
    - Resolve conflicts with alternative suggestions
    - Calculate total trip cost with breakdown
    - Save generated itinerary to agentcore_successful_itineraries
    - Create S3 snapshot of detailed itinerary
    - _Requirements: 40.1, 40.2, 40.3, 40.4, 40.5, 31.1, 33.1_
  
  - [ ] 45.10 Implement Conversation Cleanup Lambda in Go
    - Create scheduled cleanup Lambda (Go)
    - Trigger via EventBridge (daily at 2 AM)
    - Archive expired conversations to S3
    - Delete conversations older than 7 days from Aurora
    - Generate conversation summaries for completed sessions
    - Update user preference confidence scores based on new evidence
    - Extract semantic facts from archived conversations
    - Log cleanup metrics to CloudWatch
    - _Requirements: 41.1, 41.2, 41.3_
  
  - [ ] 45.11 Implement memory strategies (STM and LTM)
    - Implement Short-Term Memory (STM) strategy
      - Store last 50 messages in agentcore_messages
      - Store current conversation state in agentcore_conversation_state
      - Store pending actions (bookings awaiting confirmation)
      - Implement 24-hour TTL with 7-day grace period
    - Implement Long-Term Memory (LTM) strategies
      - User Preference Memory Strategy (namespace: /preferences/{actorId})
        - Learn preferences from conversations (cuisine, activity, budget, difficulty)
        - Store with confidence scores (0.0-1.0)
        - Update existing preferences with new evidence
      - Semantic Memory Strategy (namespace: /facts/{actorId})
        - Extract factual information (dietary restrictions, accessibility needs, travel companions)
        - Store in agentcore_semantic_memory
      - Summary Memory Strategy (namespace: /summaries/{actorId}/{sessionId})
        - Summarize completed conversations
        - Store key decisions and outcomes
        - Use for context in future conversations
    - Implement memory retrieval with relevance scoring (top_k, relevance_score thresholds)
    - Save successful itineraries to S3 with metadata
    - _Requirements: 41.1, 41.2, 41.3_
  
  - [ ] 45.12 Implement conversation state management
    - Define state schema (intent, context variables, pending actions)
    - Track current intent (trip_planning, booking_excursion, finding_restaurants, etc.)
    - Store context variables (destination, dates, time_window, cruise_info, group_composition, budget, preferences)
    - Track pending actions (booking confirmations, conflict resolutions)
    - Implement state transitions based on user messages
    - Persist state to agentcore_conversation_state table
    - Load state on each message for context continuity
    - _Requirements: 40.1, 40.2, 40.3_
  
  - [ ] 45.13 Implement AI safety and compliance features
    - Log all guardrail actions (blocks, masks) to CloudWatch
    - Implement audit trail for all AI interactions
    - Mask PII in CloudWatch logs
    - Implement rate limiting at API Gateway level (10 requests/minute per user)
    - Implement cost tracking per user (token usage, API calls)
    - Set up CloudWatch alarms for high error rates, high latency, excessive guardrail blocks
    - Implement graceful degradation when Bedrock is unavailable
    - Create fallback responses for common errors
    - _Requirements: 40.5, 22.7, 22.8, 21.8, 30.1, 30.2_
  
  - [ ]* 45.14 Write property test for memory consistency using gopter or rapid
    - **Property 18: Memory Consistency - All messages belong to valid conversations**
    - **Property 19: Preference Confidence - Confidence scores are between 0.0 and 1.0**
    - **Validates: Requirements 41.1, 41.2**
  
  - [ ]* 45.15 Write unit tests for AgentCore service using Go testing
    - Test conversation initialization and JWT validation
    - Test message processing with guardrails
    - Test input guardrail filtering (PII blocking/masking)
    - Test output guardrail filtering
    - Test prompt template selection
    - Test LLM Gateway model selection and fallback
    - Test memory loading (STM and LTM)
    - Test memory saving and state updates
    - Test preference extraction from conversations
    - Test semantic fact extraction
    - Test conversation summary generation
    - Test MCP tool execution with guardrails
    - Test itinerary generation with conflict detection
    - Test conversation cleanup and archival
    - Test error handling and fallback responses
    - _Requirements: 40.1, 40.2, 40.3, 40.5, 41.1, 41.2_
  
  - [ ]* 45.16 Write integration tests for AgentCore end-to-end flows
    - Test complete conversation flow: start → messages → memory → response
    - Test guardrail blocking of credit card numbers
    - Test guardrail masking of PII (email, phone, name)
    - Test MCP tool call flow with guardrail filtering
    - Test itinerary generation with excursion/dining/shopping queries
    - Test memory persistence across conversation sessions
    - Test preference learning from multiple conversations
    - Test conversation summary generation
    - Test S3 snapshot creation for successful itineraries
    - _Requirements: 40.1, 40.2, 40.3, 41.1, 41.2_


- [ ] 46. Checkpoint - Ensure AgentCore tests pass
  - Ensure all AgentCore tests pass (unit, integration, property-based)
  - Verify guardrails are working correctly (PII/PCI filtering)
  - Verify memory strategies are functioning (STM and LTM)
  - Verify MCP tools execute with proper guardrail filtering
  - Ask the user if questions arise.


- [ ] 47. Implement revenue tracking and analytics
  - [ ] 47.1 Create Revenue Tracking Lambda functions in Go
    - Implement trackExcursionRevenue with commission calculation (15-25%) (Go)
    - Implement trackDiningRevenue with affiliate commission (5-15%) (Go)
    - Implement trackShoppingRevenue with affiliate commission (3-10%) (Go)
    - Implement getRevenueReport by stream, destination, date range (Go)
    - Store revenue records in RDS with transaction details
    - _Requirements: 42.1, 42.2, 42.3_
  
  - [ ] 47.2 Implement analytics dashboard
    - Create Lambda in Go to aggregate revenue by stream
    - Calculate conversion rates for each revenue stream
    - Track average order value by user segment
    - Identify top-performing destinations and excursions
    - Generate revenue forecasts based on historical data
    - _Requirements: 42.1, 42.2, 42.3_
  
  - [ ] 47.3 Implement affiliate conversion tracking
    - Track affiliate link clicks with user_id and timestamp
    - Implement webhook handlers for partner conversion events
    - Match conversions to original clicks within 30-day window
    - Calculate commission amounts based on partner agreements
    - Generate partner payout reports
    - _Requirements: 42.1, 42.2, 42.3_
  
  - [ ]* 47.4 Write unit tests for revenue tracking using Go testing
    - Test commission calculation for each stream
    - Test conversion attribution
    - Test revenue aggregation
    - Test payout report generation
    - _Requirements: 42.1, 42.2, 42.3_


- [ ] 48. Update database schema for new features
  - [ ] 48.1 Create migration V002__itinerary_tables.sql
    - Create itineraries table (id, user_id, name, destination_id, start_date, end_date, notes, created_at, updated_at)
    - Create itinerary_items table (id, itinerary_id, item_type, reference_id, start_time, end_time, notes, order_index, status)
    - Create itinerary_shares table (id, itinerary_id, shared_with_user_id, permission, share_token, expires_at, created_at)
    - Create conflicts table (id, itinerary_id, item1_id, item2_id, conflict_type, resolved, created_at)
    - Add indexes on itinerary_id, user_id, item_type, start_time
    - _Requirements: 31.1, 31.6, 32.1, 33.1_
  
  - [ ] 48.2 Create migration V003__dining_shopping_tables.sql
    - Create places table (id, google_place_id, name, type, address, latitude, longitude, rating, price_level, phone, website, opening_hours, last_synced, created_at, updated_at)
    - Create restaurants table (id, place_id, cuisine_type, dietary_options, reservation_url, affiliate_link, created_at, updated_at)
    - Create shopping_locations table (id, place_id, category, products_description, affiliate_link, created_at, updated_at)
    - Create local_products table (id, shopping_location_id, name, description, price, image_url, affiliate_link, created_at)
    - Create affiliate_clicks table (id, user_id, link_type, reference_id, clicked_at, converted_at, commission_amount)
    - Add indexes on place_id, google_place_id, type, cuisine_type, category
    - _Requirements: 34.1, 35.1, 36.1, 42.1_
  
  - [ ] 48.3 Create migration V004__recommendations_tables.sql
    - Create user_segments table (id, user_id, segment_type, confidence_score, last_updated, created_at)
    - Create recommendation_scores table (id, user_id, item_type, item_id, context_score, behavior_score, popularity_score, personalization_score, total_score, created_at)
    - Create trending_items table (id, item_type, item_id, destination_id, trend_score, search_volume, created_at, updated_at)
    - Add indexes on user_id, item_type, item_id, total_score, trend_score
    - _Requirements: 37.1, 38.1, 39.1_
  
  - [ ] 48.4 Create migration V005__agentcore_tables.sql
    - Create agentcore_conversations table (conversation_id, user_id, actor_id, session_id, started_at, last_message_at, expires_at, status, metadata JSONB)
    - Create agentcore_messages table (message_id, conversation_id, role, content, tokens_used, timestamp, metadata JSONB)
    - Create agentcore_conversation_state table (state_id, conversation_id, state_key, state_value JSONB, updated_at)
    - Create agentcore_user_preferences table (preference_id, user_id, actor_id, preference_type, preference_value, confidence_score, learned_from_conversation_id, created_at, updated_at)
    - Create agentcore_semantic_memory table (memory_id, user_id, actor_id, fact_type, fact_content, confidence_score, source_conversation_id, created_at)
    - Create agentcore_conversation_summaries table (summary_id, user_id, actor_id, session_id, conversation_id, summary_text, key_decisions JSONB, created_at)
    - Create agentcore_successful_itineraries table (itinerary_id, user_id, destination_id, user_segment, itinerary_data JSONB, user_rating, created_at, s3_snapshot_key)
    - Add indexes on user_id, conversation_id, expires_at, preference_type, fact_type, session_id
    - Configure pg_cron for TTL cleanup (delete conversations older than 7 days)
    - _Requirements: 40.1, 41.1, 41.2_
  
  - [ ] 48.5 Create migration V006__supplier_info_tables.sql
    - Create supplier_info table (id, supplier_id, bank_account, tax_id, commission_rate, payment_schedule, created_at, updated_at)
    - Encrypt bank_account and tax_id fields with KMS
    - Add indexes on supplier_id
    - _Requirements: 10.1, 21.6_


- [ ] 49. Update infrastructure for new services
  - [ ] 49.1 Create agentcore-stack.ts for decoupled AgentCore service
    - Create dedicated API Gateway for AgentCore with JWT authorizer
    - Configure AWS Bedrock model access (Claude 3 Sonnet and Haiku)
    - Create AWS Bedrock Guardrail with PII/PCI filters
      - Sensitive information filters: BLOCK credit cards, MASK emails/phones/names/addresses
      - Content filters: BLOCK hate speech, insults, sexual content, violence
      - Denied topics: financial/medical/legal advice
      - Prompt attack detection enabled
    - Set up AWS Bedrock Prompt Management with versioned prompts
      - Trip planning prompts (general, cruise, flight)
      - Recommendation prompts (excursion, dining, shopping)
      - Conflict resolution prompts (detector, resolver)
    - Create Lambda functions for AgentCore:
      - agentcore-orchestrator (1024 MB, 60s timeout)
      - agentcore-memory-manager (512 MB, 30s timeout)
      - agentcore-mcp-executor (1024 MB, 30s timeout)
      - agentcore-itinerary-generator (2048 MB, 120s timeout)
      - agentcore-cleanup (256 MB, 300s timeout)
    - Configure Aurora PostgreSQL tables for AgentCore memory
      - agentcore_conversations, agentcore_messages, agentcore_conversation_state
      - agentcore_user_preferences, agentcore_semantic_memory
      - agentcore_conversation_summaries, agentcore_successful_itineraries
    - Set up S3 bucket for AgentCore memory snapshots and conversation logs
    - Configure EventBridge rule for daily cleanup (2 AM)
    - Set up IAM roles with least privilege for all Lambda functions
    - Configure CloudWatch log groups with 30-day retention
    - Set up X-Ray tracing for AgentCore requests
    - Configure VPC endpoints for Bedrock access
    - _Requirements: 40.1, 40.2, 40.5, 41.1, 41.2_
  
  - [ ] 49.2 Create analytics-stack.ts for Elasticsearch
    - Set up OpenSearch/Elasticsearch domain for analytics
    - Configure index templates for places, recommendations, trends
    - Set up Kibana for analytics dashboards
    - Configure index lifecycle policies (30-day retention)
    - Set up IAM roles for Lambda to write to Elasticsearch
    - _Requirements: 37.1, 39.1_
  
  - [ ] 49.3 Update network-stack.ts for external API access
    - Configure NAT Gateway for Lambda internet access
    - Set up VPC endpoints for AWS services (Bedrock, Secrets Manager)
    - Configure security groups for external API calls
    - Add security group for AgentCore Lambda functions
    - _Requirements: 36.1, 39.1, 40.1_
  
  - [ ] 49.4 Update secrets-stack.ts for new API keys
    - Store Google Places API key in Secrets Manager
    - Store Google Trends API credentials in Secrets Manager
    - Store affiliate partner API keys in Secrets Manager
    - Store Bedrock Guardrail ID and version in Secrets Manager
    - Configure automatic rotation for API keys (90 days)
    - _Requirements: 36.1, 39.1, 40.5, 42.1_
  
  - [ ] 49.5 Create monitoring-stack.ts for AgentCore observability
    - Create CloudWatch dashboard for AgentCore metrics
      - Conversation count, message count, token usage
      - Guardrail blocks/masks, denied topics
      - LLM latency, error rates
      - Memory retrieval latency
      - MCP tool execution metrics
    - Configure CloudWatch alarms:
      - High error rate (> 5%)
      - High latency (> 5s p99)
      - Excessive guardrail blocks (> 100/hour)
      - High token usage (> 1M tokens/day)
      - High cost (> $500/day)
    - Set up SNS topic for alarm notifications
    - Configure X-Ray service map for AgentCore
    - _Requirements: 40.5, 24.1, 24.2, 24.5_


- [ ] 50. Checkpoint - Ensure all new tests pass
  - Ensure all tests pass for new services, ask the user if questions arise.


- [ ] 51. Integration testing for new features
  - [ ]* 51.1 Write integration test for itinerary management
    - Test itinerary creation → add items → detect conflicts → share with user
    - Verify conflict detection with overlapping times
    - Verify sharing permissions (view vs edit)
    - _Requirements: 31.1, 31.6, 32.1, 33.1_
  
  - [ ]* 51.2 Write integration test for dining and shopping
    - Test Google Places sync → restaurant search → add to itinerary → generate affiliate link
    - Verify place data enrichment (photos, hours, ratings)
    - Verify affiliate click tracking
    - _Requirements: 34.1, 36.1, 42.1_
  
  - [ ]* 51.3 Write integration test for enhanced recommendations
    - Test user segmentation → calculate multi-factor scores → generate recommendations
    - Verify score weights (30% context, 25% behavior, 25% popularity, 20% personalization)
    - Verify cross-sell suggestions
    - _Requirements: 37.1, 37.2, 38.1, 42.1_
  
  - [ ]* 51.4 Write integration test for AgentCore service
    - Test conversation start → JWT authentication → send messages → guardrail filtering → LLM response
    - Test guardrail blocking of credit card numbers (PCI compliance)
    - Test guardrail masking of PII (email, phone, name, address)
    - Test MCP tool execution with guardrail filtering
    - Test itinerary generation: query excursions → query dining → query shopping → generate optimized itinerary
    - Test memory persistence: STM (last 50 messages) → LTM (preferences, facts, summaries)
    - Test preference learning across multiple conversations
    - Test semantic fact extraction (dietary restrictions, accessibility needs)
    - Test conversation summary generation
    - Test S3 snapshot creation for successful itineraries
    - Test conversation cleanup and archival (7-day retention)
    - Test prompt template selection based on context
    - Test LLM model fallback (Sonnet → Haiku on rate limits)
    - Test streaming responses for real-time interaction
    - Test cost tracking (token usage per user)
    - _Requirements: 40.1, 40.2, 40.3, 40.5, 41.1, 41.2, 41.3_
  
  - [ ]* 51.5 Write integration test for revenue tracking
    - Test excursion booking → dining affiliate click → shopping conversion → revenue report
    - Verify commission calculations for all three streams
    - Verify conversion attribution within 30-day window
    - _Requirements: 42.1, 42.2, 42.3_
  
  - [ ]* 51.6 Write integration test for AgentCore guardrails
    - Test denied topics (financial/medical/legal advice)
    - Test content filtering (hate speech, insults, sexual content, violence)
    - Test prompt attack detection (jailbreak, injection)
    - Test PII masking in responses
    - Test guardrail metrics logging to CloudWatch
    - _Requirements: 40.5, 22.7, 22.8, 21.8_


- [ ] 52. Performance optimization for new services
  - [ ] 52.1 Optimize Elasticsearch queries
    - Create optimized index mappings for places and recommendations
    - Implement query result caching (5 minutes)
    - Configure index sharding based on destination
    - Tune relevance scoring for search quality
    - _Requirements: 36.1, 37.1, NFR-1_
  
  - [ ] 52.2 Optimize Google Places API usage
    - Implement aggressive caching (24 hours for place details)
    - Batch place sync operations (100 places per batch)
    - Implement incremental sync (only changed places)
    - Use Place ID for efficient lookups
    - Monitor API quota usage and implement throttling
    - _Requirements: 36.1, 36.2, NFR-1_
  
  - [ ] 52.3 Optimize AgentCore performance
    - Implement streaming responses for better UX (Server-Sent Events)
    - Enable prompt caching for system prompts (50% cost reduction)
    - Optimize context window size (last 50 messages, ~100K tokens)
    - Use provisioned concurrency for AgentCore Orchestrator Lambda (reduce cold starts)
    - Implement request batching for memory updates
    - Cache user preferences and facts in Lambda memory (5-minute TTL)
    - Use Claude 3 Haiku for simple queries, Sonnet for complex (cost optimization)
    - Implement connection pooling for Aurora PostgreSQL (RDS Proxy)
    - Optimize memory retrieval queries with proper indexes
    - Implement lazy loading for LTM (only load when needed)
    - _Requirements: 40.1, 40.2, 41.1, NFR-1, NFR-2_
  
  - [ ] 52.4 Optimize recommendation engine
    - Pre-calculate recommendation scores for popular items
    - Cache user segments (24 hours)
    - Implement lazy loading for recommendation details
    - Use Elasticsearch aggregations for fast scoring
    - _Requirements: 37.1, 37.2, NFR-1_
  
  - [ ] 52.5 Optimize AgentCore costs
    - Monitor token usage per user and set alerts
    - Implement token budget limits per conversation (max 100K tokens)
    - Use prompt caching to reduce repeated system prompt costs
    - Implement model selection based on query complexity
    - Track and optimize MCP tool call frequency
    - Set up cost allocation tags for AgentCore resources
    - _Requirements: 40.1, 40.2, NFR-2_


- [ ] 53. Final validation and deployment of new features
  - Run full test suite including new services (unit, integration, property-based)
  - Perform load testing for AgentCore service
    - Test concurrent conversations (100 users)
    - Test streaming response performance
    - Test memory retrieval latency under load
    - Test guardrail performance with high request volume
  - Validate AWS Bedrock integration
    - Test Claude 3 Sonnet and Haiku model access
    - Verify prompt caching is working (check cost reduction)
    - Test model fallback (Sonnet → Haiku)
    - Verify streaming responses work correctly
  - Validate AWS Bedrock Guardrails
    - Test PII/PCI filtering with sample data
    - Verify credit card numbers are blocked
    - Verify emails, phones, names, addresses are masked
    - Test denied topics (financial/medical/legal advice)
    - Test content filtering (hate speech, insults, etc.)
    - Test prompt attack detection
    - Verify guardrail metrics are logged to CloudWatch
  - Validate Prompt Management
    - Test all prompt templates load correctly
    - Verify prompt versioning works
    - Test prompt selection based on context
  - Validate MCP tool integration
    - Test all MCP tools execute correctly
    - Verify guardrail filtering on tool inputs/outputs
    - Test tool error handling
  - Validate memory management
    - Test STM persistence (last 50 messages)
    - Test LTM strategies (preferences, facts, summaries)
    - Verify TTL cleanup works (7-day retention)
    - Test S3 snapshot creation
    - Verify memory retrieval with relevance scoring
  - Validate Google Places API integration and rate limits
  - Deploy new database migrations to staging
    - AgentCore tables (conversations, messages, state, preferences, memory, summaries, itineraries)
    - Itinerary tables
    - Dining and shopping tables
    - Recommendations tables
  - Deploy new Lambda functions to staging
    - AgentCore Lambdas (orchestrator, memory manager, MCP executor, itinerary generator, cleanup)
    - Itinerary service Lambdas
    - Dining and shopping service Lambdas
    - Enhanced recommendation engine Lambdas
  - Deploy AgentCore Gateway API to staging
  - Configure AWS Bedrock Guardrails in staging
  - Configure AWS Bedrock Prompt Management in staging
  - Perform user acceptance testing for new features
    - Test conversational trip planning
    - Test itinerary generation with conflict detection
    - Test dining and shopping recommendations
    - Test enhanced multi-factor recommendations
    - Test PII masking in responses
  - Monitor CloudWatch metrics for new services
    - AgentCore: conversations, messages, token usage, guardrail blocks, latency, errors
    - Recommendations: score calculations, user segments, cross-sell conversions
    - Revenue: commission tracking, affiliate conversions
  - Validate cost tracking and alerts
    - Verify token usage tracking per user
    - Verify cost allocation tags
    - Test cost threshold alarms
  - Deploy to production with gradual rollout
    - Deploy AgentCore service (10% → 50% → 100%)
    - Deploy other new services
    - Monitor for errors and performance issues
  - _Requirements: NFR-1, NFR-2, NFR-3, NFR-5, 40.1, 40.2, 40.5, 41.1, 41.2_
