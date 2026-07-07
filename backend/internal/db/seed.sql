-- Dev-only seed data. Fixed UUIDs + ON CONFLICT DO NOTHING keep it idempotent.
-- Applied at startup when APP_SEED=true (docker-compose sets this).

INSERT INTO teams (id, name, slug) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Platform', 'platform'),
    ('22222222-2222-2222-2222-222222222222', 'Payments', 'payments')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, entra_oid, email, name) VALUES
    ('aaaaaaaa-0000-0000-0000-000000000001', NULL, 'dev.admin@example.com', 'Dev Admin'),
    ('aaaaaaaa-0000-0000-0000-000000000002', NULL, 'dev.editor@example.com', 'Dev Editor')
ON CONFLICT (id) DO NOTHING;

INSERT INTO team_members (team_id, user_id, role) VALUES
    ('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-0000-0000-0000-000000000001', 'ADMIN'),
    ('22222222-2222-2222-2222-222222222222', 'aaaaaaaa-0000-0000-0000-000000000002', 'EDITOR')
ON CONFLICT (team_id, user_id) DO NOTHING;

INSERT INTO services (id, team_id, name, slug, description, repo_url, runbook_url, lifecycle) VALUES
    ('33333333-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111',
     'API Gateway', 'api-gateway',
     'Edge gateway routing external traffic to internal services.',
     'https://github.com/example/api-gateway', 'https://runbooks.example.com/api-gateway', 'production'),
    ('33333333-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111',
     'Identity Service', 'identity-service',
     'Issues and validates tokens; integrates with Entra ID.',
     'https://github.com/example/identity-service', NULL, 'production'),
    ('33333333-0000-0000-0000-000000000003', '22222222-2222-2222-2222-222222222222',
     'Payments API', 'payments-api',
     'Card and invoice payment processing.',
     'https://github.com/example/payments-api', 'https://runbooks.example.com/payments-api', 'production'),
    ('33333333-0000-0000-0000-000000000004', '22222222-2222-2222-2222-222222222222',
     'Refunds Worker', 'refunds-worker',
     'Async worker handling refund batches.',
     'https://github.com/example/refunds-worker', NULL, 'beta'),
    ('33333333-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111',
     'Legacy Reports', 'legacy-reports',
     'Old reporting stack, scheduled for decommissioning.',
     NULL, NULL, 'deprecated')
ON CONFLICT (id) DO NOTHING;

INSERT INTO tags (id, name) VALUES
    ('44444444-0000-0000-0000-000000000001', 'go'),
    ('44444444-0000-0000-0000-000000000002', 'typescript'),
    ('44444444-0000-0000-0000-000000000003', 'edge'),
    ('44444444-0000-0000-0000-000000000004', 'pci')
ON CONFLICT (id) DO NOTHING;

INSERT INTO service_tags (service_id, tag_id) VALUES
    ('33333333-0000-0000-0000-000000000001', '44444444-0000-0000-0000-000000000001'),
    ('33333333-0000-0000-0000-000000000001', '44444444-0000-0000-0000-000000000003'),
    ('33333333-0000-0000-0000-000000000002', '44444444-0000-0000-0000-000000000001'),
    ('33333333-0000-0000-0000-000000000003', '44444444-0000-0000-0000-000000000001'),
    ('33333333-0000-0000-0000-000000000003', '44444444-0000-0000-0000-000000000004'),
    ('33333333-0000-0000-0000-000000000004', '44444444-0000-0000-0000-000000000002')
ON CONFLICT (service_id, tag_id) DO NOTHING;
