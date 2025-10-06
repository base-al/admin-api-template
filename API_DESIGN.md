# ISP Billing System - API Design Guide

## Endpoint Structure

### Customer-Centric Endpoints (Customer Portal/Dashboard)
These endpoints are ideal for customer-facing operations:

```http
# View customer's active plans
GET    /api/customers/:id/plans

# Assign a new plan to customer (purchase/subscribe)
POST   /api/customers/:id/plans
Body: {
  "plan_id": 1,
  "payment_amount": 25.99
}

# Cancel/remove a plan from customer
DELETE /api/customers/:id/plans/:planId

# Get customer's billing history
GET    /api/customers/:id/billing-history

# Renew customer's specific plan
POST   /api/customers/:id/plans/:planId/renew
```

### Administrative Endpoints (Admin Dashboard)
These endpoints are for managing the relationship entity:

```http
# List all customer-plan assignments with filtering
GET    /api/customer-plans?status=active&expiring_soon=true

# Get detailed assignment information
GET    /api/customer-plans/:id

# Update assignment (change status, extend expiry)
PUT    /api/customer-plans/:id
Body: {
  "status": "suspended",
  "expires_at": "2025-10-06"
}

# Delete assignment record
DELETE /api/customer-plans/:id

# Bulk operations
POST   /api/customer-plans/bulk-suspend
POST   /api/customer-plans/bulk-activate
GET    /api/customer-plans/expiring?days=7
POST   /api/customer-plans/process-expired
```

## Why This Dual Approach?

### 1. **Clear Intent**
- `/customers/:id/plans` - "I want to do something with THIS customer's plans"
- `/customer-plans` - "I want to manage plan assignments across the system"

### 2. **Permission Scoping**
```go
// Customer endpoint - customer can only access their own data
if customerId != authenticatedUser.CustomerId {
    return Unauthorized
}

// Admin endpoint - requires admin role
if !user.HasRole("admin") {
    return Forbidden
}
```

### 3. **Different Use Cases**
- **Customer Portal**: Uses `/customers/:id/plans` for self-service
- **Admin Dashboard**: Uses `/customer-plans` for system-wide management
- **Billing Cron Jobs**: Uses `/customer-plans/expiring` for automation

### 4. **Better Caching**
- Customer-specific endpoints can be cached per customer
- Admin endpoints can have different cache strategies

## Implementation Example

### Customer Controller (`app/customers/controller.go`)
```go
// Customer-focused operations
func (c *Controller) GetCustomerPlans(ctx *router.Context) error {
    customerId := ctx.Param("id")
    // Returns plans with customer context
}

func (c *Controller) AssignPlanToCustomer(ctx *router.Context) error {
    customerId := ctx.Param("id")
    // Assigns plan with customer validation
}
```

### CustomerPlans Controller (`app/customer_plans/controller.go`)
```go
// Administrative operations
func (c *Controller) ListAllAssignments(ctx *router.Context) error {
    // Returns all assignments with filtering
}

func (c *Controller) ProcessExpiredPlans(ctx *router.Context) error {
    // Bulk process expired plans
}
```

## Real-World Usage Examples

### Customer Self-Service Flow
```javascript
// Customer viewing their plans
const response = await fetch(`/api/customers/${customerId}/plans`, {
    headers: { 'Authorization': `Bearer ${customerToken}` }
});

// Customer purchasing a new plan
await fetch(`/api/customers/${customerId}/plans`, {
    method: 'POST',
    body: JSON.stringify({ plan_id: selectedPlanId, payment_amount: 29.99 })
});
```

### Admin Management Flow
```javascript
// Admin viewing all expiring plans
const expiring = await fetch('/api/customer-plans/expiring?days=7', {
    headers: { 'X-Api-Key': apiKey }
});

// Admin bulk suspending overdue accounts
await fetch('/api/customer-plans/bulk-suspend', {
    method: 'POST',
    body: JSON.stringify({ customer_ids: overdueCustomerIds })
});
```

## Recommended for Your ISP System

Given your ISP billing requirements, implement:

1. **Customer Endpoints** (Priority: High)
   - `GET /api/customers/:id/plans` - View customer's plans
   - `POST /api/customers/:id/plans` - Subscribe to plan
   - `DELETE /api/customers/:id/plans/:planId` - Cancel subscription

2. **Admin Endpoints** (Priority: High)
   - `GET /api/customer-plans` - List all assignments
   - `GET /api/customer-plans/expiring` - Monitor expiring plans
   - `PUT /api/customer-plans/:id` - Update assignment status

3. **Future Enhancements**
   - `POST /api/customer-plans/auto-renew` - Automated renewals
   - `GET /api/customer-plans/revenue-report` - Financial reporting
   - `POST /api/customers/:id/plans/:planId/upgrade` - Plan upgrades

This design provides flexibility, clarity, and scalability for your ISP billing system.