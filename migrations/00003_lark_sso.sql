-- +goose Up
ALTER TABLE public.users ADD COLUMN lark_open_id character varying(255);
CREATE UNIQUE INDEX users_lark_open_id_unique ON public.users (lark_open_id);

-- +goose Down
DROP INDEX public.users_lark_open_id_unique;
ALTER TABLE public.users DROP COLUMN lark_open_id;
