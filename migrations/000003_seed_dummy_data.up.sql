-- Seed Customers
INSERT INTO customers (id, name, email, phone) 
VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Customer', 'test@example.com', '081234567890')
ON CONFLICT (id) DO NOTHING;

-- Seed Payment Methods (Common Midtrans Methods)
INSERT INTO payment_methods (id, code, name, type, is_active, config) 
VALUES 
    -- E-Wallets & QRIS
    ('550e8400-e29b-41d4-a716-446655440000', 'qris', 'QRIS (Gopay/OVO/Dana/LinkAja)', 'E-WALLET', true, '{"provider": "midtrans", "acquirer": "gopay"}'),
    ('550e8400-e29b-41d4-a716-446655440002', 'gopay', 'GoPay', 'E-WALLET', true, '{"provider": "midtrans"}'),
    ('550e8400-e29b-41d4-a716-446655440003', 'shopeepay', 'ShopeePay', 'E-WALLET', true, '{"provider": "midtrans"}'),
    
    -- Virtual Accounts (Bank Transfer)
    ('550e8400-e29b-41d4-a716-446655440004', 'bca_va', 'BCA Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "bca"}'),
    ('550e8400-e29b-41d4-a716-446655440005', 'bni_va', 'BNI Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "bni"}'),
    ('550e8400-e29b-41d4-a716-446655440006', 'echannel', 'Mandiri Bill Payment', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "mandiri"}'),
    ('550e8400-e29b-41d4-a716-446655440010', 'bri_va', 'BRI Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "bri"}'),
    ('550e8400-e29b-41d4-a716-446655440011', 'permata_va', 'Permata Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "permata"}'),
    
    -- Credit Card
    ('550e8400-e29b-41d4-a716-446655440007', 'credit_card', 'Credit Card (Visa/Mastercard/JCB)', 'CREDIT_CARD', true, '{"provider": "midtrans", "3ds": true}'),
    
    -- Over The Counter (Convenience Stores)
    ('550e8400-e29b-41d4-a716-446655440008', 'alfamart', 'Alfamart / Alfamidi / Dan+Dan', 'OTC', true, '{"provider": "midtrans", "store": "alfamart"}'),
    ('550e8400-e29b-41d4-a716-446655440009', 'indomaret', 'Indomaret', 'OTC', true, '{"provider": "midtrans", "store": "indomaret"}'),
    
    -- Cardless Credit / Paylater
    ('550e8400-e29b-41d4-a716-446655440012', 'akulaku', 'Akulaku PayLater', 'PAYLATER', true, '{"provider": "midtrans"}')
ON CONFLICT (id) DO NOTHING;
