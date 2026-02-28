ALTER TABLE payments
    DROP CONSTRAINT IF EXISTS payments_status_check;

UPDATE payments
SET status = 'paid'
WHERE status = 'completed';

ALTER TABLE payments
    ADD CONSTRAINT payments_status_check
        CHECK (status IN ('placeholder', 'paid', 'refunded'));

CREATE INDEX IF NOT EXISTS idx_bookings_coach_schedule_active
    ON bookings(coach_id, scheduled_at)
    WHERE status <> 'cancelled';

CREATE INDEX IF NOT EXISTS idx_payments_booking_id
    ON payments(booking_id);
