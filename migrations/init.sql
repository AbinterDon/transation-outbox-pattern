-- Create the orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create the outbox table
CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL, -- correlates to order_id
    payload JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for polling performance
CREATE INDEX IF NOT EXISTS idx_outbox_status_created_at ON outbox (status, created_at);

-- Consumer side: idempotency tracking
CREATE TABLE IF NOT EXISTS processed_messages (
    message_id UUID PRIMARY KEY,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
