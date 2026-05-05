DELETE FROM payment_methods WHERE id IN (
    'qris', 'gopay', 'shopeepay', 'bca_va', 'bni_va', 'echannel', 
    'bri_va', 'permata_va', 'credit_card', 'alfamart', 'indomaret', 'akulaku'
);

DELETE FROM customers WHERE id = '550e8400-e29b-41d4-a716-446655440001';
