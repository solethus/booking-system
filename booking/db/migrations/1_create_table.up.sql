CREATE  TABLE booking (
    id BIGSERIAL PRIMARY KEY,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    email TEXT NOT NULL,
    create_at TIMESTAMP NOT NULL DEFAULT NOW()
);