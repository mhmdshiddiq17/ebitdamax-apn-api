-- +goose Up

CREATE TABLE public.customer_analyses (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    full_name character varying(255) NOT NULL,
    occupation_role character varying(50) NOT NULL,
    occupation_other character varying(255),
    age smallint NOT NULL,
    gender character varying(20) NOT NULL,
    interview_purpose text NOT NULL,
    summary text NOT NULL,
    sentiment smallint NOT NULL DEFAULT 3,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);

CREATE INDEX customer_analyses_user_id_created_at_index
    ON public.customer_analyses (user_id, created_at DESC);

-- +goose Down

DROP TABLE public.customer_analyses;
