CREATE SCHEMA IF NOT EXISTS department;

CREATE TABLE IF NOT EXISTS department.department (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    resources JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS department.user_department (
    user_id UUID NOT NULL REFERENCES account.user(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES department.department(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, department_id)
);

CREATE INDEX IF NOT EXISTS idx_user_department_user_id ON department.user_department(user_id);
CREATE INDEX IF NOT EXISTS idx_user_department_department_id ON department.user_department(department_id);

