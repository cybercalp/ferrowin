-- ============================================================
-- Seed: Entidades (Clientes, Proveedores, Ambos)
-- Empresa: Ferrowin S.L. (00000000-0000-4000-a000-000000000001)
-- Sector: Ferretería industrial
-- ============================================================

-- Truncate in dependency order
TRUNCATE entidad_notas, entidad_contactos, entidad_direcciones, entidades RESTART IDENTITY CASCADE;

-- ============================================================
-- CLIENTES SOLO
-- ============================================================
INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, activo, roles) VALUES
  ('e1000001-0000-4000-a000-000000000001', '00000000-0000-4000-a000-000000000001',
   'Construcciones Navarro S.L.',    'B12345678', 'compras@navarro-sl.es',     '+34 963 112 233', TRUE,  'CLIENTE'),
  ('e1000001-0000-4000-a000-000000000002', '00000000-0000-4000-a000-000000000001',
   'Reformas García e Hijos',        'B23456789', 'admin@reformasgarcia.com',  '+34 91 234 5678',  TRUE,  'CLIENTE'),
  ('e1000001-0000-4000-a000-000000000003', '00000000-0000-4000-a000-000000000001',
   'Instalaciones López & Cía',      'B34567890', 'pedidos@instal-lopez.es',   '+34 93 456 7890',  TRUE,  'CLIENTE'),
  ('e1000001-0000-4000-a000-000000000004', '00000000-0000-4000-a000-000000000001',
   'Electricidad Martínez S.A.',     'A45678901', 'compras@electricidad-mtz.com','+34 95 567 8901', TRUE,  'CLIENTE'),
  ('e1000001-0000-4000-a000-000000000005', '00000000-0000-4000-a000-000000000001',
   'Fontanería Ruiz Hermanos',       'B56789012', 'fontaneria.ruiz@gmail.com', '+34 96 678 9012',  TRUE,  'CLIENTE'),
  ('e1000001-0000-4000-a000-000000000006', '00000000-0000-4000-a000-000000000001',
   'Talleres Mecánicos Sánchez',     'B67890123', 'taller@mecanicossanchez.es','+34 97 789 0123',  TRUE,  'CLIENTE'),
  ('e1000001-0000-4000-a000-000000000007', '00000000-0000-4000-a000-000000000001',
   'Obras y Proyectos Fernández',    'B78901234', 'info@opfernandez.es',       '+34 98 890 1234',  TRUE,  'CLIENTE'),
  ('e1000001-0000-4000-a000-000000000008', '00000000-0000-4000-a000-000000000001',
   'Pinturas y Acabados Díez S.L.',  'B89012345', 'pedidos@pinturadiez.com',   '+34 91 901 2345',  FALSE, 'CLIENTE');

-- ============================================================
-- PROVEEDORES SOLO
-- ============================================================
INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, activo, roles) VALUES
  ('e2000001-0000-4000-a000-000000000001', '00000000-0000-4000-a000-000000000001',
   'Hilti Ibérica S.A.',             'A11111111', 'pedidos.es@hilti.com',       '+34 900 100 700', TRUE, 'PROVEEDOR'),
  ('e2000001-0000-4000-a000-000000000002', '00000000-0000-4000-a000-000000000001',
   'Stanley Black & Decker España',  'A22222222', 'ventas@stanleyes.com',       '+34 93 200 1234', TRUE, 'PROVEEDOR'),
  ('e2000001-0000-4000-a000-000000000003', '00000000-0000-4000-a000-000000000001',
   'Bosch Herramientas S.L.',        'B33333333', 'b2b@bosch-es.com',           '+34 91 300 2345', TRUE, 'PROVEEDOR'),
  ('e2000001-0000-4000-a000-000000000004', '00000000-0000-4000-a000-000000000001',
   'Makita España S.A.',             'A44444444', 'pedidos@makita.es',          '+34 91 400 3456', TRUE, 'PROVEEDOR'),
  ('e2000001-0000-4000-a000-000000000005', '00000000-0000-4000-a000-000000000001',
   'Würth España S.A.',              'A55555555', 'clientes@wuerth.es',         '+34 91 500 4567', TRUE, 'PROVEEDOR');

-- ============================================================
-- CLIENTES + PROVEEDORES (AMBOS)
-- ============================================================
INSERT INTO entidades (id, empresa_id, razon_social, nif, email, telefono, activo, roles) VALUES
  ('e3000001-0000-4000-a000-000000000001', '00000000-0000-4000-a000-000000000001',
   'Distribuciones Metálicas Vega',  'B66666666', 'comercial@vega-metalicas.es','+34 96 600 5678', TRUE, 'CLIENTE,PROVEEDOR'),
  ('e3000001-0000-4000-a000-000000000002', '00000000-0000-4000-a000-000000000001',
   'Ferretería Industrial Pedraza',  'B77777777', 'info@ferreteriapedraza.es',  '+34 91 700 6789', TRUE, 'CLIENTE,PROVEEDOR');

-- ============================================================
-- DIRECCIONES
-- ============================================================
INSERT INTO entidad_direcciones (id, entidad_id, tipo_direccion, calle, ciudad, provincia, codigo_postal, pais) VALUES
  -- Construcciones Navarro
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000001', 'FISCAL',  'Calle Mayor 14, 3º A',       'Valencia',    'Valencia',   '46001', 'España'),
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000001', 'ENVIO',   'Polígono Can Tunis, Nave 8', 'Valencia',    'Valencia',   '46012', 'España'),
  -- Reformas García
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000002', 'FISCAL',  'Avda. de la Constitución 22','Madrid',      'Madrid',     '28001', 'España'),
  -- Instalaciones López
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000003', 'FISCAL',  'Gran Via 87, 1º B',          'Barcelona',   'Barcelona',  '08008', 'España'),
  -- Hilti
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000001', 'FISCAL',  'C/ Julián Camarillo 4',      'Madrid',      'Madrid',     '28037', 'España'),
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000001', 'ENVIO',   'Pol. Ind. Cobo Calleja s/n', 'Fuenlabrada', 'Madrid',     '28947', 'España'),
  -- Bosch
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000003', 'FISCAL',  'Ctra. de Fuencarral 40',     'Madrid',      'Madrid',     '28029', 'España'),
  -- Würth
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000005', 'FISCAL',  'Autovía A-2 Km 23',          'Alcalá de Henares', 'Madrid', '28806', 'España'),
  -- Distribuciones Metálicas Vega (ambos)
  (gen_random_uuid(), 'e3000001-0000-4000-a000-000000000001', 'FISCAL',  'Polígono Industrial El Olivar, C/A 12', 'Sagunto', 'Valencia', '46500', 'España'),
  (gen_random_uuid(), 'e3000001-0000-4000-a000-000000000001', 'ENVIO',   'Muelle 3, Puerto de Sagunto', 'Sagunto',   'Valencia',   '46520', 'España'),
  -- Ferretería Industrial Pedraza (ambos)
  (gen_random_uuid(), 'e3000001-0000-4000-a000-000000000002', 'FISCAL',  'Plaza del Dos de Mayo 5',    'Madrid',      'Madrid',     '28004', 'España');

-- ============================================================
-- CONTACTOS
-- ============================================================
INSERT INTO entidad_contactos (id, entidad_id, nombre, puesto, email, telefono) VALUES
  -- Construcciones Navarro
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000001', 'Alberto Navarro Soler',   'Gerente',           'a.navarro@navarro-sl.es',  '+34 609 111 001'),
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000001', 'Carmen Pons Ibáñez',      'Jefa de Compras',   'c.pons@navarro-sl.es',     '+34 609 111 002'),
  -- Reformas García
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000002', 'Miguel Ángel García',     'Propietario',       'mgarcia@reformasgarcia.com','+34 620 222 001'),
  -- Hilti
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000001', 'Laura Jiménez Torres',    'Responsable B2B',   'l.jimenez@hilti.com',      '+34 630 333 001'),
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000001', 'Ramón Esteve Planells',   'Técnico Comercial', 'r.esteve@hilti.com',       '+34 630 333 002'),
  -- Bosch
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000003', 'Ignacio Reyes Medina',    'Key Account Mgr',   'i.reyes@bosch.es',         '+34 640 444 001'),
  -- Distribuciones Metálicas Vega
  (gen_random_uuid(), 'e3000001-0000-4000-a000-000000000001', 'Sergi Vega Palomino',     'Director Comercial','s.vega@vega-metalicas.es', '+34 650 555 001'),
  (gen_random_uuid(), 'e3000001-0000-4000-a000-000000000001', 'Eva Martín Quijada',      'Logística',         'e.martin@vega-metalicas.es','+34 650 555 002');

-- ============================================================
-- NOTAS
-- ============================================================
INSERT INTO entidad_notas (id, entidad_id, nota, creado_en) VALUES
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000001',
   'Cliente habitual con pago a 30 días. Descuento pactado del 5% en tornillería. Revisar condiciones en próxima renovación.',
   NOW() - INTERVAL '30 days'),
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000001',
   'Solicita siempre albarán firmado. Enviar factura antes del día 20 de cada mes.',
   NOW() - INTERVAL '5 days'),
  (gen_random_uuid(), 'e1000001-0000-4000-a000-000000000002',
   'Pago al contado. Muy puntual. Interesado en catálogo de herramientas eléctricas para el Q3.',
   NOW() - INTERVAL '15 days'),
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000001',
   'Proveedor principal de taladros y fijaciones. Plazo de entrega 48h para stock habitual, 5 días para especiales.',
   NOW() - INTERVAL '20 days'),
  (gen_random_uuid(), 'e2000001-0000-4000-a000-000000000001',
   'Rappel anual del 3% si superamos 60.000€. Contactar con Laura antes de agosto para revisar volumen.',
   NOW() - INTERVAL '7 days'),
  (gen_random_uuid(), 'e3000001-0000-4000-a000-000000000001',
   'Acuerdo de colaboración recíproco: nos compran accesorios de montaje, nosotros les compramos perfiles de acero.',
   NOW() - INTERVAL '45 days'),
  (gen_random_uuid(), 'e3000001-0000-4000-a000-000000000001',
   'Pendiente de firmar el contrato marco 2026. Sergi Vega lo tiene en revisión con su gestor.',
   NOW() - INTERVAL '3 days');
