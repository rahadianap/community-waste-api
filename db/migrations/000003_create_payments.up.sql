CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed');

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    waste_id UUID NOT NULL REFERENCES waste_pickups(id) ON DELETE CASCADE,
    amount NUMERIC(12, 2) NOT NULL,
    payment_date TIMESTAMPTZ NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    proof_file_url VARCHAR(255) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payments_household_id ON payments(household_id);
CREATE INDEX IF NOT EXISTS idx_payments_waste_id ON payments(waste_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_household_pending ON payments(household_id, status) WHERE status = 'pending';
