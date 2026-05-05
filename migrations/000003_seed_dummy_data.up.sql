-- Seed Customers
INSERT INTO customers (id, name, email, phone) 
VALUES ('550e8400-e29b-41d4-a716-446655440001', 'Test Customer', 'test@example.com', '081234567890')
ON CONFLICT (id) DO NOTHING;

-- Seed Payment Methods (Common Midtrans Methods)
INSERT INTO payment_methods (id, name, type, is_active, config) 
VALUES 
    -- E-Wallets & QRIS
    ('qris', 'QRIS (Gopay/OVO/Dana/LinkAja)', 'E-WALLET', true, '{"provider": "midtrans", "acquirer": "gopay"}'),
    ('gopay', 'GoPay', 'E-WALLET', true, '{"provider": "midtrans"}'),
    ('shopeepay', 'ShopeePay', 'E-WALLET', true, '{"provider": "midtrans"}'),
    
    -- Virtual Accounts (Bank Transfer)
    ('bca_va', 'BCA Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "bca"}'),
    ('bni_va', 'BNI Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "bni"}'),
    ('echannel', 'Mandiri Bill Payment', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "mandiri"}'),
    ('bri_va', 'BRI Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "bri"}'),
    ('permata_va', 'Permata Virtual Account', 'VIRTUAL_ACCOUNT', true, '{"provider": "midtrans", "bank": "permata"}'),
    
    -- Credit Card
    ('credit_card', 'Credit Card (Visa/Mastercard/JCB)', 'CREDIT_CARD', true, '{"provider": "midtrans", "3ds": true}'),
    
    -- Over The Counter (Convenience Stores)
    ('alfamart', 'Alfamart / Alfamidi / Dan+Dan', 'OTC', true, '{"provider": "midtrans", "store": "alfamart"}'),
    ('indomaret', 'Indomaret', 'OTC', true, '{"provider": "midtrans", "store": "indomaret"}'),
    
    -- Cardless Credit / Paylater
    ('akulaku', 'Akulaku PayLater', 'PAYLATER', true, '{"provider": "midtrans"}')
ON CONFLICT (id) DO NOTHING;
