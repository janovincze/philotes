-- Philotes Sample E-Commerce Database Seed Data
-- This script populates the database with realistic sample data for testing CDC.

-- ============================================================================
-- CUSTOMERS (100 records)
-- ============================================================================
INSERT INTO customers (id, first_name, last_name, email, phone, metadata) VALUES
    ('00000001-0000-0000-0000-000000000001', 'Alice', 'Johnson', 'alice.johnson@example.com', '+1-555-0101', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('00000001-0000-0000-0000-000000000002', 'Bob', 'Smith', 'bob.smith@example.com', '+1-555-0102', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('00000001-0000-0000-0000-000000000003', 'Carol', 'Williams', 'carol.williams@example.com', '+1-555-0103', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('00000001-0000-0000-0000-000000000004', 'David', 'Brown', 'david.brown@example.com', '+1-555-0104', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('00000001-0000-0000-0000-000000000005', 'Emma', 'Davis', 'emma.davis@example.com', '+1-555-0105', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('00000001-0000-0000-0000-000000000006', 'Frank', 'Miller', 'frank.miller@example.com', '+1-555-0106', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('00000001-0000-0000-0000-000000000007', 'Grace', 'Wilson', 'grace.wilson@example.com', '+1-555-0107', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('00000001-0000-0000-0000-000000000008', 'Henry', 'Moore', 'henry.moore@example.com', '+1-555-0108', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('00000001-0000-0000-0000-000000000009', 'Iris', 'Taylor', 'iris.taylor@example.com', '+1-555-0109', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000010', 'Jack', 'Anderson', 'jack.anderson@example.com', '+1-555-0110', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000011', 'Kate', 'Thomas', 'kate.thomas@example.com', '+1-555-0111', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000012', 'Liam', 'Jackson', 'liam.jackson@example.com', '+1-555-0112', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000013', 'Mia', 'White', 'mia.white@example.com', '+1-555-0113', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000014', 'Noah', 'Harris', 'noah.harris@example.com', '+1-555-0114', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000015', 'Olivia', 'Martin', 'olivia.martin@example.com', '+1-555-0115', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000016', 'Peter', 'Garcia', 'peter.garcia@example.com', '+1-555-0116', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000017', 'Quinn', 'Martinez', 'quinn.martinez@example.com', '+1-555-0117', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000018', 'Rachel', 'Robinson', 'rachel.robinson@example.com', '+1-555-0118', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000019', 'Sam', 'Clark', 'sam.clark@example.com', '+1-555-0119', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000020', 'Tina', 'Rodriguez', 'tina.rodriguez@example.com', '+1-555-0120', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000021', 'Uma', 'Lewis', 'uma.lewis@example.com', '+1-555-0121', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000022', 'Victor', 'Lee', 'victor.lee@example.com', '+1-555-0122', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000023', 'Wendy', 'Walker', 'wendy.walker@example.com', '+1-555-0123', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000024', 'Xavier', 'Hall', 'xavier.hall@example.com', '+1-555-0124', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000025', 'Yolanda', 'Allen', 'yolanda.allen@example.com', '+1-555-0125', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000026', 'Zach', 'Young', 'zach.young@example.com', '+1-555-0126', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000027', 'Amy', 'King', 'amy.king@example.com', '+1-555-0127', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000028', 'Brian', 'Wright', 'brian.wright@example.com', '+1-555-0128', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000029', 'Cindy', 'Scott', 'cindy.scott@example.com', '+1-555-0129', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000030', 'Derek', 'Green', 'derek.green@example.com', '+1-555-0130', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000031', 'Elena', 'Adams', 'elena.adams@example.com', '+1-555-0131', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000032', 'Felix', 'Baker', 'felix.baker@example.com', '+1-555-0132', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000033', 'Gina', 'Nelson', 'gina.nelson@example.com', '+1-555-0133', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000034', 'Hugo', 'Hill', 'hugo.hill@example.com', '+1-555-0134', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000035', 'Ivy', 'Ramirez', 'ivy.ramirez@example.com', '+1-555-0135', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000036', 'James', 'Campbell', 'james.campbell@example.com', '+1-555-0136', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000037', 'Kelly', 'Mitchell', 'kelly.mitchell@example.com', '+1-555-0137', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000038', 'Leo', 'Roberts', 'leo.roberts@example.com', '+1-555-0138', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000039', 'Maya', 'Carter', 'maya.carter@example.com', '+1-555-0139', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000040', 'Nick', 'Phillips', 'nick.phillips@example.com', '+1-555-0140', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000041', 'Olive', 'Evans', 'olive.evans@example.com', '+1-555-0141', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000042', 'Paul', 'Turner', 'paul.turner@example.com', '+1-555-0142', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000043', 'Rosa', 'Torres', 'rosa.torres@example.com', '+1-555-0143', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000044', 'Steve', 'Parker', 'steve.parker@example.com', '+1-555-0144', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000045', 'Tara', 'Collins', 'tara.collins@example.com', '+1-555-0145', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000046', 'Ulrich', 'Edwards', 'ulrich.edwards@example.com', '+1-555-0146', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000047', 'Vera', 'Stewart', 'vera.stewart@example.com', '+1-555-0147', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}'),
    ('c0000001-0000-0000-0000-000000000048', 'Will', 'Sanchez', 'will.sanchez@example.com', '+1-555-0148', '{"loyalty_tier": "bronze", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000049', 'Xena', 'Morris', 'xena.morris@example.com', '+1-555-0149', '{"loyalty_tier": "gold", "preferences": {"newsletter": true}}'),
    ('c0000001-0000-0000-0000-000000000050', 'Yuri', 'Rogers', 'yuri.rogers@example.com', '+1-555-0150', '{"loyalty_tier": "silver", "preferences": {"newsletter": false}}');

-- Generate additional 50 customers using a pattern
INSERT INTO customers (first_name, last_name, email, phone, metadata)
SELECT
    'Customer' || i,
    'User' || i,
    'customer' || i || '@example.com',
    '+1-555-' || LPAD(i::text, 4, '0'),
    jsonb_build_object(
        'loyalty_tier', CASE WHEN i % 3 = 0 THEN 'gold' WHEN i % 3 = 1 THEN 'silver' ELSE 'bronze' END,
        'preferences', jsonb_build_object('newsletter', i % 2 = 0)
    )
FROM generate_series(51, 100) AS i;

-- ============================================================================
-- PRODUCTS (50 records)
-- ============================================================================
INSERT INTO products (id, name, sku, description, price, cost, inventory_count, category, attributes) VALUES
    ('00000001-0000-0000-0001-000000000001', 'Wireless Bluetooth Headphones', 'WBH-001', 'Premium noise-canceling wireless headphones with 30-hour battery life', 149.99, 75.00, 150, 'Electronics', '{"brand": "AudioTech", "color": "black", "wireless": true, "battery_hours": 30}'),
    ('00000001-0000-0000-0001-000000000002', 'USB-C Charging Cable 2m', 'UCC-002', 'Fast charging USB-C cable, 2 meters, braided nylon', 19.99, 5.00, 500, 'Electronics', '{"length_meters": 2, "material": "braided_nylon", "fast_charge": true}'),
    ('00000001-0000-0000-0001-000000000003', 'Ergonomic Office Chair', 'EOC-003', 'Adjustable lumbar support office chair with mesh back', 299.99, 150.00, 45, 'Furniture', '{"material": "mesh", "adjustable_height": true, "armrests": true, "max_weight_kg": 120}'),
    ('00000001-0000-0000-0001-000000000004', 'Stainless Steel Water Bottle', 'SWB-004', 'Insulated 750ml water bottle, keeps drinks cold 24h or hot 12h', 29.99, 10.00, 300, 'Kitchen', '{"capacity_ml": 750, "insulated": true, "material": "stainless_steel"}'),
    ('00000001-0000-0000-0001-000000000005', 'LED Desk Lamp', 'LDL-005', 'Dimmable LED desk lamp with USB charging port', 49.99, 20.00, 200, 'Electronics', '{"dimmable": true, "usb_port": true, "color_temperature": "adjustable"}'),
    ('00000001-0000-0000-0001-000000000006', 'Cotton T-Shirt', 'CTS-006', '100% organic cotton t-shirt, unisex fit', 24.99, 8.00, 1000, 'Clothing', '{"material": "organic_cotton", "sizes": ["S", "M", "L", "XL"], "colors": ["white", "black", "navy"]}'),
    ('00000001-0000-0000-0001-000000000007', 'Running Shoes', 'RSH-007', 'Lightweight running shoes with memory foam insole', 89.99, 40.00, 120, 'Sports', '{"brand": "SpeedRun", "sizes": [7, 8, 9, 10, 11, 12], "cushioning": "memory_foam"}'),
    ('00000001-0000-0000-0001-000000000008', 'Yoga Mat', 'YGM-008', 'Non-slip yoga mat, 6mm thickness, eco-friendly', 34.99, 12.00, 250, 'Sports', '{"thickness_mm": 6, "material": "eco_rubber", "non_slip": true}'),
    ('00000001-0000-0000-0001-000000000009', 'French Press Coffee Maker', 'FPC-009', 'Borosilicate glass french press, 1 liter capacity', 39.99, 15.00, 180, 'Kitchen', '{"capacity_liters": 1, "material": "borosilicate_glass", "dishwasher_safe": true}'),
    ('00000001-0000-0000-0001-000000000010', 'Mechanical Keyboard', 'MKB-010', 'RGB mechanical keyboard with Cherry MX switches', 129.99, 60.00, 80, 'Electronics', '{"switches": "cherry_mx_blue", "rgb": true, "layout": "full_size", "wireless": false}'),
    ('00000001-0000-0000-0001-000000000011', 'Wireless Mouse', 'WMS-011', 'Ergonomic wireless mouse with adjustable DPI', 59.99, 25.00, 220, 'Electronics', '{"dpi_range": [800, 1600, 3200], "ergonomic": true, "battery_type": "rechargeable"}'),
    ('00000001-0000-0000-0001-000000000012', 'Laptop Stand', 'LST-012', 'Aluminum laptop stand with adjustable height', 44.99, 18.00, 150, 'Electronics', '{"material": "aluminum", "adjustable": true, "compatible_sizes": "10-17 inch"}'),
    ('00000001-0000-0000-0001-000000000013', 'Backpack', 'BPK-013', 'Water-resistant laptop backpack with USB charging port', 69.99, 30.00, 100, 'Bags', '{"capacity_liters": 25, "laptop_compartment": "15.6 inch", "water_resistant": true}'),
    ('00000001-0000-0000-0001-000000000014', 'Wireless Earbuds', 'WEB-014', 'True wireless earbuds with active noise cancellation', 179.99, 80.00, 90, 'Electronics', '{"anc": true, "battery_hours": 8, "water_resistant": "IPX4"}'),
    ('00000001-0000-0000-0001-000000000015', 'Smart Watch', 'SWT-015', 'Fitness tracking smartwatch with heart rate monitor', 199.99, 90.00, 60, 'Electronics', '{"heart_rate": true, "gps": true, "water_resistant": "5ATM", "battery_days": 7}'),
    ('00000001-0000-0000-0001-000000000016', 'Electric Kettle', 'EKT-016', '1.7L electric kettle with temperature control', 54.99, 22.00, 140, 'Kitchen', '{"capacity_liters": 1.7, "temperature_control": true, "auto_shutoff": true}'),
    ('00000001-0000-0000-0001-000000000017', 'Plant Pot Set', 'PPS-017', 'Set of 3 ceramic plant pots with drainage', 32.99, 12.00, 200, 'Home', '{"quantity": 3, "material": "ceramic", "drainage": true, "sizes": ["small", "medium", "large"]}'),
    ('00000001-0000-0000-0001-000000000018', 'Notebook Set', 'NBS-018', 'Set of 5 lined notebooks, A5 size', 14.99, 4.00, 400, 'Office', '{"quantity": 5, "size": "A5", "pages_each": 100, "ruled": true}'),
    ('00000001-0000-0000-0001-000000000019', 'Desk Organizer', 'DSO-019', 'Bamboo desk organizer with multiple compartments', 27.99, 10.00, 180, 'Office', '{"material": "bamboo", "compartments": 6, "phone_holder": true}'),
    ('00000001-0000-0000-0001-000000000020', 'Wall Clock', 'WCK-020', 'Silent non-ticking wall clock, modern design', 24.99, 8.00, 120, 'Home', '{"diameter_cm": 30, "silent": true, "style": "modern"}'),
    ('00000001-0000-0000-0001-000000000021', 'Resistance Bands Set', 'RBS-021', 'Set of 5 resistance bands with different strengths', 22.99, 7.00, 300, 'Sports', '{"quantity": 5, "resistance_levels": ["extra_light", "light", "medium", "heavy", "extra_heavy"]}'),
    ('00000001-0000-0000-0001-000000000022', 'Portable Charger', 'PCH-022', '20000mAh portable charger with fast charging', 49.99, 20.00, 160, 'Electronics', '{"capacity_mah": 20000, "fast_charge": true, "ports": 2}'),
    ('00000001-0000-0000-0001-000000000023', 'Sunglasses', 'SNG-023', 'Polarized sunglasses with UV400 protection', 79.99, 25.00, 100, 'Accessories', '{"polarized": true, "uv_protection": "UV400", "frame_material": "acetate"}'),
    ('00000001-0000-0000-0001-000000000024', 'Towel Set', 'TWS-024', 'Set of 4 cotton bath towels', 44.99, 18.00, 150, 'Home', '{"quantity": 4, "material": "100% cotton", "sizes": ["bath", "hand"]}'),
    ('00000001-0000-0000-0001-000000000025', 'Candle Set', 'CDS-025', 'Set of 3 scented soy candles', 29.99, 10.00, 200, 'Home', '{"quantity": 3, "material": "soy_wax", "scents": ["vanilla", "lavender", "ocean_breeze"]}'),
    ('00000001-0000-0000-0001-000000000026', 'HDMI Cable 3m', 'HDM-026', 'High-speed HDMI 2.1 cable, 3 meters', 24.99, 8.00, 350, 'Electronics', '{"length_meters": 3, "version": "2.1", "max_resolution": "8K"}'),
    ('00000001-0000-0000-0001-000000000027', 'Mouse Pad XL', 'MPD-027', 'Extended mouse pad for gaming, 900x400mm', 19.99, 6.00, 250, 'Electronics', '{"width_mm": 900, "height_mm": 400, "thickness_mm": 4, "surface": "smooth"}'),
    ('00000001-0000-0000-0001-000000000028', 'Webcam HD', 'WCM-028', '1080p HD webcam with built-in microphone', 69.99, 30.00, 90, 'Electronics', '{"resolution": "1080p", "fps": 30, "microphone": true, "autofocus": true}'),
    ('00000001-0000-0000-0001-000000000029', 'Desk Fan', 'DSF-029', 'USB desk fan with 3 speed settings', 18.99, 6.00, 280, 'Electronics', '{"power": "USB", "speeds": 3, "oscillation": false}'),
    ('00000001-0000-0000-0001-000000000030', 'Lunch Box', 'LBX-030', 'Stainless steel bento lunch box with compartments', 26.99, 10.00, 200, 'Kitchen', '{"material": "stainless_steel", "compartments": 3, "leak_proof": true}'),
    ('00000001-0000-0000-0001-000000000031', 'Umbrella', 'UMB-031', 'Automatic folding umbrella, windproof', 22.99, 8.00, 180, 'Accessories', '{"automatic": true, "windproof": true, "size": "compact"}'),
    ('00000001-0000-0000-0001-000000000032', 'Bluetooth Speaker', 'BTS-032', 'Portable Bluetooth speaker with 12h battery', 79.99, 35.00, 110, 'Electronics', '{"battery_hours": 12, "waterproof": "IPX7", "power_watts": 20}'),
    ('00000001-0000-0000-0001-000000000033', 'Reading Light', 'RLT-033', 'Clip-on LED reading light, rechargeable', 14.99, 5.00, 320, 'Electronics', '{"clip_on": true, "rechargeable": true, "brightness_levels": 3}'),
    ('00000001-0000-0000-0001-000000000034', 'Throw Pillow', 'TPW-034', 'Decorative throw pillow, 45x45cm', 19.99, 7.00, 250, 'Home', '{"size_cm": "45x45", "material": "polyester", "removable_cover": true}'),
    ('00000001-0000-0000-0001-000000000035', 'Kitchen Scale', 'KSC-035', 'Digital kitchen scale with tare function', 24.99, 9.00, 170, 'Kitchen', '{"max_weight_kg": 5, "precision_g": 1, "tare_function": true}'),
    ('00000001-0000-0000-0001-000000000036', 'Travel Mug', 'TMG-036', 'Insulated travel mug, 500ml', 21.99, 8.00, 280, 'Kitchen', '{"capacity_ml": 500, "insulated": true, "leak_proof": true}'),
    ('00000001-0000-0000-0001-000000000037', 'Wireless Charger', 'WCG-037', '15W fast wireless charger', 34.99, 12.00, 200, 'Electronics', '{"power_watts": 15, "fast_charge": true, "compatible": ["Qi"]}'),
    ('00000001-0000-0000-0001-000000000038', 'Photo Frame Set', 'PFS-038', 'Set of 6 photo frames, various sizes', 39.99, 15.00, 130, 'Home', '{"quantity": 6, "material": "wood", "sizes": ["4x6", "5x7", "8x10"]}'),
    ('00000001-0000-0000-0001-000000000039', 'Cutting Board Set', 'CBS-039', 'Set of 3 bamboo cutting boards', 34.99, 14.00, 150, 'Kitchen', '{"quantity": 3, "material": "bamboo", "sizes": ["small", "medium", "large"]}'),
    ('00000001-0000-0000-0001-000000000040', 'Hand Cream Set', 'HCS-040', 'Set of 4 moisturizing hand creams', 18.99, 6.00, 220, 'Beauty', '{"quantity": 4, "volume_ml_each": 30, "scents": ["rose", "jasmine", "coconut", "unscented"]}'),
    ('00000001-0000-0000-0001-000000000041', 'Monitor Stand', 'MST-041', 'Adjustable monitor stand with storage', 39.99, 16.00, 100, 'Electronics', '{"adjustable": true, "storage_drawer": true, "max_weight_kg": 20}'),
    ('00000001-0000-0000-0001-000000000042', 'Fitness Tracker', 'FTR-042', 'Basic fitness tracker with step counter', 49.99, 20.00, 140, 'Electronics', '{"step_counter": true, "sleep_tracking": true, "water_resistant": true}'),
    ('00000001-0000-0000-0001-000000000043', 'Coaster Set', 'CST-043', 'Set of 6 cork coasters', 12.99, 4.00, 300, 'Home', '{"quantity": 6, "material": "cork", "diameter_cm": 10}'),
    ('00000001-0000-0000-0001-000000000044', 'Pen Set', 'PNS-044', 'Set of 12 gel pens, assorted colors', 9.99, 3.00, 400, 'Office', '{"quantity": 12, "type": "gel", "colors": "assorted"}'),
    ('00000001-0000-0000-0001-000000000045', 'Extension Cord', 'EXC-045', '3m extension cord with 4 outlets and USB', 22.99, 9.00, 180, 'Electronics', '{"length_meters": 3, "outlets": 4, "usb_ports": 2, "surge_protection": true}'),
    ('00000001-0000-0000-0001-000000000046', 'Shoe Rack', 'SRK-046', '4-tier shoe rack, holds 12 pairs', 29.99, 12.00, 90, 'Home', '{"tiers": 4, "capacity_pairs": 12, "material": "metal_plastic"}'),
    ('00000001-0000-0000-0001-000000000047', 'Blender', 'BLD-047', 'Personal blender for smoothies, 600ml', 39.99, 16.00, 100, 'Kitchen', '{"capacity_ml": 600, "power_watts": 300, "speeds": 2, "portable_cup": true}'),
    ('00000001-0000-0000-0001-000000000048', 'Wallet', 'WLT-048', 'Leather bifold wallet with RFID blocking', 44.99, 18.00, 120, 'Accessories', '{"material": "genuine_leather", "rfid_blocking": true, "card_slots": 8}'),
    ('00000001-0000-0000-0001-000000000049', 'Dumbbell Set', 'DBS-049', 'Adjustable dumbbell set, 2-10kg each', 89.99, 40.00, 60, 'Sports', '{"weight_range_kg": "2-10", "adjustable": true, "quantity": 2}'),
    ('00000001-0000-0000-0001-000000000050', 'Essential Oil Set', 'EOS-050', 'Set of 6 essential oils for aromatherapy', 29.99, 10.00, 180, 'Beauty', '{"quantity": 6, "oils": ["lavender", "eucalyptus", "peppermint", "tea_tree", "lemon", "orange"]}');

-- ============================================================================
-- ORDERS (500 records)
-- ============================================================================

-- First, let's create a helper function to generate random addresses
CREATE OR REPLACE FUNCTION random_address()
RETURNS JSONB AS $$
DECLARE
    streets TEXT[] := ARRAY['Main St', 'Oak Ave', 'Maple Dr', 'Cedar Ln', 'Pine Rd', 'Elm St', 'Park Ave', 'Lake Dr', 'Hill Rd', 'River Way'];
    cities TEXT[] := ARRAY['New York', 'Los Angeles', 'Chicago', 'Houston', 'Phoenix', 'Philadelphia', 'San Antonio', 'San Diego', 'Dallas', 'San Jose'];
    states TEXT[] := ARRAY['NY', 'CA', 'IL', 'TX', 'AZ', 'PA', 'TX', 'CA', 'TX', 'CA'];
    idx INT;
BEGIN
    idx := floor(random() * 10)::int + 1;
    RETURN jsonb_build_object(
        'street', (floor(random() * 999) + 1)::text || ' ' || streets[idx],
        'city', cities[idx],
        'state', states[idx],
        'zip', LPAD((floor(random() * 99999)::int)::text, 5, '0'),
        'country', 'US'
    );
END;
$$ LANGUAGE plpgsql;

-- Generate 500 orders
INSERT INTO orders (customer_id, order_number, status, subtotal, tax, shipping, total, shipping_address, billing_address, ordered_at)
SELECT
    -- Random customer from first 100
    (SELECT id FROM customers OFFSET floor(random() * 100) LIMIT 1),
    'ORD-' || LPAD(i::text, 6, '0'),
    (ARRAY['pending', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled'])[floor(random() * 6)::int + 1],
    round((random() * 500 + 20)::numeric, 2) AS subtotal,
    0, -- Will be calculated
    round((random() * 15 + 5)::numeric, 2) AS shipping,
    0, -- Will be calculated
    random_address(),
    random_address(),
    CURRENT_TIMESTAMP - (random() * interval '365 days')
FROM generate_series(1, 500) AS i;

-- Update tax and total
UPDATE orders SET
    tax = round(subtotal * 0.08, 2),
    total = round(subtotal + (subtotal * 0.08) + shipping, 2);

-- ============================================================================
-- ORDER_ITEMS (~2000 records, 1-6 items per order)
-- ============================================================================
INSERT INTO order_items (order_id, product_id, quantity, unit_price, discount)
SELECT
    o.id AS order_id,
    (SELECT id FROM products OFFSET floor(random() * 50) LIMIT 1) AS product_id,
    floor(random() * 3 + 1)::int AS quantity,
    (SELECT price FROM products OFFSET floor(random() * 50) LIMIT 1) AS unit_price,
    CASE WHEN random() < 0.2 THEN round((random() * 10)::numeric, 2) ELSE 0 END AS discount
FROM orders o
CROSS JOIN generate_series(1, floor(random() * 5 + 1)::int) AS item_num;

-- Clean up helper function
DROP FUNCTION IF EXISTS random_address();

-- ============================================================================
-- VERIFY DATA
-- ============================================================================
DO $$
DECLARE
    customer_count INT;
    product_count INT;
    order_count INT;
    item_count INT;
BEGIN
    SELECT COUNT(*) INTO customer_count FROM customers;
    SELECT COUNT(*) INTO product_count FROM products;
    SELECT COUNT(*) INTO order_count FROM orders;
    SELECT COUNT(*) INTO item_count FROM order_items;

    RAISE NOTICE 'Seed data loaded successfully:';
    RAISE NOTICE '  - Customers: %', customer_count;
    RAISE NOTICE '  - Products: %', product_count;
    RAISE NOTICE '  - Orders: %', order_count;
    RAISE NOTICE '  - Order Items: %', item_count;
END $$;
