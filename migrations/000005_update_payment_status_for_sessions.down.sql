DROP INDEX IF EXISTS idx_payments_booking_id;

DROP INDEX IF EXISTS idx_bookings_coach_schedule_active;

ALTER TABLE payments
    DROP CONSTRAINT IF EXISTS payments_status_check;

UPDATE payments
SET status = 'completed'
WHERE status = 'paid';

ALTER TABLE payments
    ADD CONSTRAINT payments_status_check
        CHECK (status IN ('placeholder', 'completed', 'refunded'));
