-- Seed Customers
INSERT INTO customers (id, name, email, phone) 
VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Customer', 'test@example.com', '081234567890')
ON CONFLICT (id) DO NOTHING;

-- Seed Payment Methods
INSERT INTO payment_methods (id, code, name, type, is_active, config) 
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'midtrans_qris', 'QRIS (Midtrans)', 'QRIS', true, '{"provider": "midtrans"}')
ON CONFLICT (id) DO NOTHING;
