-- +goose Up
--
-- PostgreSQL database dump
--


-- Dumped from database version 17.11
-- Dumped by pg_dump version 18.4 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: business_process_steps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.business_process_steps (
    id bigint NOT NULL,
    business_process_id bigint NOT NULL,
    sequence smallint NOT NULL,
    process_group character varying(255) NOT NULL,
    detail_process text NOT NULL,
    pic character varying(255) NOT NULL,
    standard_time_minutes smallint NOT NULL,
    output_target text,
    responsibility_value smallint,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: business_process_steps_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.business_process_steps_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: business_process_steps_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.business_process_steps_id_seq OWNED BY public.business_process_steps.id;


--
-- Name: business_processes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.business_processes (
    id bigint NOT NULL,
    code character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    unit_name character varying(255),
    unit_code character varying(255),
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    user_id bigint
);


--
-- Name: business_processes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.business_processes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: business_processes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.business_processes_id_seq OWNED BY public.business_processes.id;


--
-- Name: cache; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cache (
    key character varying(255) NOT NULL,
    value text NOT NULL,
    expiration bigint NOT NULL
);


--
-- Name: cache_locks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cache_locks (
    key character varying(255) NOT NULL,
    owner character varying(255) NOT NULL,
    expiration bigint NOT NULL
);


--
-- Name: ebitda_values; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ebitda_values (
    id bigint NOT NULL,
    organization_id bigint NOT NULL,
    period_date date,
    year smallint NOT NULL,
    scenario character varying(255) NOT NULL,
    revenue numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    doc_variable numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    doc_fixed numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    ioc numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    toc numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    ebitda numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    ebitda_margin numeric(10,4),
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    excel_import_id bigint,
    source_sheet character varying(255),
    raw_payload json,
    classification character varying(255),
    man_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    method_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    material_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    machine_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL
);


--
-- Name: ebitda_values_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ebitda_values_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ebitda_values_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ebitda_values_id_seq OWNED BY public.ebitda_values.id;


--
-- Name: ebitdamax_kdkmp; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ebitdamax_kdkmp (
    id bigint NOT NULL,
    sdm_kdkmp_entry_id bigint NOT NULL,
    report_date date NOT NULL,
    target_revenue character varying(255),
    actual_revenue character varying(255),
    actual_cost character varying(255),
    total_duration character varying(255),
    performance_scoring character varying(255),
    created_by bigint,
    updated_by bigint,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    plan_revenue character varying(255),
    target_cost character varying(255),
    plan_cost character varying(255),
    target_ebitda character varying(255),
    plan_ebitda character varying(255),
    actual_ebitda character varying(255),
    target_ebitda_margin character varying(255),
    actual_ebitda_margin character varying(255),
    plan_revenue_requires_review boolean DEFAULT false NOT NULL,
    operational_attendance json,
    operational_attendance_saved_at timestamp(0) without time zone,
    selected_task_ids json,
    actual_variable_cost character varying(255)
);


--
-- Name: ebitdamax_kdkmp_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ebitdamax_kdkmp_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ebitdamax_kdkmp_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ebitdamax_kdkmp_id_seq OWNED BY public.ebitdamax_kdkmp.id;


--
-- Name: excel_imports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.excel_imports (
    id bigint NOT NULL,
    filename character varying(255) NOT NULL,
    original_filename character varying(255) NOT NULL,
    status character varying(255) DEFAULT 'pending'::character varying NOT NULL,
    total_rows integer DEFAULT 0 NOT NULL,
    success_rows integer DEFAULT 0 NOT NULL,
    failed_rows integer DEFAULT 0 NOT NULL,
    created_by bigint,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: excel_imports_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.excel_imports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: excel_imports_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.excel_imports_id_seq OWNED BY public.excel_imports.id;


--
-- Name: failed_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.failed_jobs (
    id bigint NOT NULL,
    uuid character varying(255) NOT NULL,
    connection character varying(255) NOT NULL,
    queue character varying(255) NOT NULL,
    payload text NOT NULL,
    exception text NOT NULL,
    failed_at timestamp(0) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: failed_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.failed_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: failed_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.failed_jobs_id_seq OWNED BY public.failed_jobs.id;


--
-- Name: import_error_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.import_error_logs (
    id bigint NOT NULL,
    excel_import_id bigint NOT NULL,
    row_number integer,
    sheet_name character varying(255),
    message text NOT NULL,
    payload json,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: import_error_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.import_error_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: import_error_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.import_error_logs_id_seq OWNED BY public.import_error_logs.id;


--
-- Name: job_batches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.job_batches (
    id character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    total_jobs integer NOT NULL,
    pending_jobs integer NOT NULL,
    failed_jobs integer NOT NULL,
    failed_job_ids text NOT NULL,
    options text,
    cancelled_at integer,
    created_at integer NOT NULL,
    finished_at integer
);


--
-- Name: jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.jobs (
    id bigint NOT NULL,
    queue character varying(255) NOT NULL,
    payload text NOT NULL,
    attempts smallint NOT NULL,
    reserved_at integer,
    available_at integer NOT NULL,
    created_at integer NOT NULL
);


--
-- Name: jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.jobs_id_seq OWNED BY public.jobs.id;


--
-- Name: koperasi_sarpras_status_points; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.koperasi_sarpras_status_points (
    id bigint NOT NULL,
    nik character varying(255),
    nama_koperasi character varying(255),
    provinsi character varying(255),
    kota_kabupaten character varying(255),
    kecamatan character varying(255),
    desa character varying(255),
    kodim character varying(255),
    lat numeric(10,6) NOT NULL,
    lng numeric(10,6) NOT NULL,
    validation_status character varying(255),
    progress_percentage numeric(5,1) DEFAULT '0'::numeric NOT NULL,
    batch character varying(255),
    completed_sarpras_count integer DEFAULT 0 NOT NULL,
    sarpras_less_than_6 boolean DEFAULT true NOT NULL,
    sarpras_primary_lengkap boolean DEFAULT false NOT NULL,
    sarpras_secondary_lengkap boolean DEFAULT false NOT NULL,
    sarpras_lengkap boolean DEFAULT false NOT NULL,
    has_po boolean DEFAULT false NOT NULL,
    has_receipt boolean DEFAULT false NOT NULL,
    has_sales boolean DEFAULT false NOT NULL,
    synced_at timestamp(0) without time zone,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: koperasi_sarpras_status_points_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.koperasi_sarpras_status_points_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: koperasi_sarpras_status_points_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.koperasi_sarpras_status_points_id_seq OWNED BY public.koperasi_sarpras_status_points.id;


--
-- Name: meeting_minute_attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.meeting_minute_attachments (
    id bigint NOT NULL,
    meeting_minute_id bigint NOT NULL,
    disk character varying(255) DEFAULT 'local'::character varying NOT NULL,
    path character varying(255) NOT NULL,
    original_name character varying(255) NOT NULL,
    mime_type character varying(255),
    size bigint DEFAULT '0'::bigint NOT NULL,
    uploaded_by bigint,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: meeting_minute_attachments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.meeting_minute_attachments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: meeting_minute_attachments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.meeting_minute_attachments_id_seq OWNED BY public.meeting_minute_attachments.id;


--
-- Name: meeting_minute_item_status_histories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.meeting_minute_item_status_histories (
    id bigint NOT NULL,
    meeting_minute_item_id bigint NOT NULL,
    from_status character varying(255) NOT NULL,
    to_status character varying(255) NOT NULL,
    note text,
    changed_by bigint,
    changed_by_name character varying(255) NOT NULL,
    created_at timestamp(0) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: meeting_minute_item_status_histories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.meeting_minute_item_status_histories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: meeting_minute_item_status_histories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.meeting_minute_item_status_histories_id_seq OWNED BY public.meeting_minute_item_status_histories.id;


--
-- Name: meeting_minute_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.meeting_minute_items (
    id bigint NOT NULL,
    meeting_minute_id bigint NOT NULL,
    subject character varying(255) NOT NULL,
    description text,
    action text,
    objectives text,
    date_start date,
    date_finish date,
    pic character varying(255),
    status character varying(255) DEFAULT 'open'::character varying NOT NULL,
    remarks text,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: meeting_minute_items_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.meeting_minute_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: meeting_minute_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.meeting_minute_items_id_seq OWNED BY public.meeting_minute_items.id;


--
-- Name: meeting_minutes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.meeting_minutes (
    id bigint NOT NULL,
    title character varying(255) NOT NULL,
    meeting_date date NOT NULL,
    start_time time(0) without time zone,
    end_time time(0) without time zone,
    location character varying(255),
    attendees text,
    created_by bigint,
    updated_by bigint,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: meeting_minutes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.meeting_minutes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: meeting_minutes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.meeting_minutes_id_seq OWNED BY public.meeting_minutes.id;


--
-- Name: migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.migrations (
    id integer NOT NULL,
    migration character varying(255) NOT NULL,
    batch integer NOT NULL
);


--
-- Name: migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.migrations_id_seq OWNED BY public.migrations.id;


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id uuid NOT NULL,
    type character varying(255) NOT NULL,
    notifiable_type character varying(255) NOT NULL,
    notifiable_id bigint NOT NULL,
    data text NOT NULL,
    read_at timestamp(0) without time zone,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: organization_calculations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_calculations (
    id bigint NOT NULL,
    organization_id bigint NOT NULL,
    source_sheet character varying(255) DEFAULT 'Kalkulasi'::character varying NOT NULL,
    classification character varying(255),
    man_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    method_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    material_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    machine_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    total_cost numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    doc_variable numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    doc_fixed numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    ioc numeric(20,2) DEFAULT '0'::numeric NOT NULL,
    raw_payload json,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: organization_calculations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.organization_calculations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: organization_calculations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.organization_calculations_id_seq OWNED BY public.organization_calculations.id;


--
-- Name: organization_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_profiles (
    id bigint NOT NULL,
    organization_id bigint NOT NULL,
    excel_import_id bigint,
    source_sheet character varying(255),
    job_description text,
    qualification text,
    value_chain text,
    method_cost numeric(20,2),
    raw_payload json,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: organization_profiles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.organization_profiles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: organization_profiles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.organization_profiles_id_seq OWNED BY public.organization_profiles.id;


--
-- Name: organizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizations (
    id bigint NOT NULL,
    parent_id bigint,
    code character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    level character varying(255),
    directorate_group character varying(255),
    is_revenue_center boolean DEFAULT false NOT NULL,
    is_cost_center boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    slug character varying(255),
    depth smallint DEFAULT '0'::smallint NOT NULL,
    path character varying(255),
    node_type character varying(255),
    is_active boolean DEFAULT true NOT NULL
);


--
-- Name: organizations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.organizations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: organizations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.organizations_id_seq OWNED BY public.organizations.id;


--
-- Name: passkeys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.passkeys (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    credential_id character varying(255) NOT NULL,
    credential json NOT NULL,
    last_used_at timestamp(0) without time zone,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: passkeys_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.passkeys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: passkeys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.passkeys_id_seq OWNED BY public.passkeys.id;


--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.password_reset_tokens (
    email character varying(255) NOT NULL,
    token character varying(255) NOT NULL,
    created_at timestamp(0) without time zone
);


--
-- Name: plan_ebitda_matrices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_ebitda_matrices (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    business_process_id bigint,
    unit_cost_assumption_id bigint,
    revenue_plan_id bigint,
    code character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    source_sheet character varying(255) NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: plan_ebitda_matrices_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_ebitda_matrices_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_ebitda_matrices_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.plan_ebitda_matrices_id_seq OWNED BY public.plan_ebitda_matrices.id;


--
-- Name: plan_ebitda_matrix_processes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_ebitda_matrix_processes (
    id bigint NOT NULL,
    plan_ebitda_matrix_id bigint NOT NULL,
    business_process_step_id bigint,
    sequence smallint NOT NULL,
    process_group character varying(255) NOT NULL,
    detail_process text NOT NULL,
    unit_name character varying(255),
    pic character varying(255) NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: plan_ebitda_matrix_processes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_ebitda_matrix_processes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_ebitda_matrix_processes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.plan_ebitda_matrix_processes_id_seq OWNED BY public.plan_ebitda_matrix_processes.id;


--
-- Name: plan_ebitda_matrix_rows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_ebitda_matrix_rows (
    id bigint NOT NULL,
    plan_ebitda_matrix_id bigint NOT NULL,
    unit_cost_assumption_row_id bigint,
    section_code character varying(10) NOT NULL,
    sort_order smallint NOT NULL,
    row_type character varying(255) NOT NULL,
    label text NOT NULL,
    "values" json NOT NULL,
    total character varying(255),
    notes text,
    notes_tone character varying(255),
    is_calculated boolean DEFAULT false NOT NULL,
    source_page smallint NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: plan_ebitda_matrix_rows_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_ebitda_matrix_rows_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_ebitda_matrix_rows_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.plan_ebitda_matrix_rows_id_seq OWNED BY public.plan_ebitda_matrix_rows.id;


--
-- Name: revenue_plan_rows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.revenue_plan_rows (
    id bigint NOT NULL,
    revenue_plan_id bigint NOT NULL,
    sort_order smallint NOT NULL,
    row_type character varying(255) NOT NULL,
    display_number smallint,
    revenue_service text,
    planned_volume numeric(12,3),
    unit character varying(255),
    rate numeric(20,2),
    planned_revenue numeric(20,2),
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: revenue_plan_rows_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.revenue_plan_rows_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: revenue_plan_rows_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.revenue_plan_rows_id_seq OWNED BY public.revenue_plan_rows.id;


--
-- Name: revenue_plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.revenue_plans (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    code character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    plan_date date NOT NULL,
    rka_revenue_target numeric(20,2),
    planned_production_quantity numeric(12,3),
    days_per_month smallint NOT NULL,
    daily_rka_revenue_target numeric(20,2),
    planned_total_daily_revenue numeric(20,2) NOT NULL,
    source_sheet character varying(255) NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: revenue_plans_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.revenue_plans_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: revenue_plans_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.revenue_plans_id_seq OWNED BY public.revenue_plans.id;


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    id bigint NOT NULL,
    uuid uuid NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    level character varying(255) NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    domain character varying(16) DEFAULT 'apn'::character varying NOT NULL
);


--
-- Name: roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.roles_id_seq OWNED BY public.roles.id;


--
-- Name: sdm_kdkmp_entries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sdm_kdkmp_entries (
    id bigint NOT NULL,
    nama_koperasi character varying(255) NOT NULL,
    jumlah_karyawan integer DEFAULT 0 NOT NULL,
    catatan text,
    created_by bigint,
    updated_by bigint,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    nik character varying(255),
    nama_kodam character varying(255),
    nama_korem character varying(255),
    nama_kodim character varying(255),
    desa character varying(255),
    kecamatan character varying(255),
    kota_kabupaten character varying(255),
    batch character varying(255),
    provinsi character varying(255)
);


--
-- Name: sdm_kdkmp_entries_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sdm_kdkmp_entries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sdm_kdkmp_entries_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sdm_kdkmp_entries_id_seq OWNED BY public.sdm_kdkmp_entries.id;


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    id character varying(255) NOT NULL,
    user_id bigint,
    ip_address character varying(45),
    user_agent text,
    payload text NOT NULL,
    last_activity integer NOT NULL
);


--
-- Name: task_additional_fields; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_additional_fields (
    id bigint NOT NULL,
    uuid uuid NOT NULL,
    task_id bigint NOT NULL,
    label character varying(255) NOT NULL,
    field_name character varying(255) NOT NULL,
    input_type character varying(255) NOT NULL,
    show_when character varying(255) NOT NULL,
    is_required boolean DEFAULT false NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    options json,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: task_additional_fields_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_additional_fields_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_additional_fields_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_additional_fields_id_seq OWNED BY public.task_additional_fields.id;


--
-- Name: task_categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_categories (
    id bigint NOT NULL,
    uuid uuid NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    description text,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: task_categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_categories_id_seq OWNED BY public.task_categories.id;


--
-- Name: task_report_values; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_report_values (
    id bigint NOT NULL,
    uuid uuid NOT NULL,
    task_report_id bigint NOT NULL,
    task_additional_field_id bigint NOT NULL,
    value text,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: task_report_values_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_report_values_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_report_values_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_report_values_id_seq OWNED BY public.task_report_values.id;


--
-- Name: task_reports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_reports (
    id bigint NOT NULL,
    uuid uuid NOT NULL,
    task_id bigint NOT NULL,
    user_id bigint NOT NULL,
    started_photo character varying(255),
    finished_photo character varying(255),
    started_at timestamp(0) without time zone,
    finished_at timestamp(0) without time zone,
    duration_minutes integer,
    status character varying(255) DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    period_key character varying(255),
    started_documents json,
    finished_documents json,
    member_allocations json,
    manager_self_assigned boolean DEFAULT false NOT NULL
);


--
-- Name: task_reports_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_reports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_reports_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_reports_id_seq OWNED BY public.task_reports.id;


--
-- Name: task_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_roles (
    id bigint NOT NULL,
    task_id bigint NOT NULL,
    role_id bigint NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: task_roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_roles_id_seq OWNED BY public.task_roles.id;


--
-- Name: tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tasks (
    id bigint NOT NULL,
    uuid uuid NOT NULL,
    task_category_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    time_require integer NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    period character varying(255) DEFAULT 'once'::character varying NOT NULL,
    lower_time_threshold_minutes integer,
    upper_time_threshold_minutes integer,
    sort_order integer,
    execution_time time(0) without time zone,
    bmc_status character varying(255) DEFAULT 'belum_dipetakan'::character varying NOT NULL,
    is_mandatory boolean DEFAULT false NOT NULL,
    fixed_cost jsonb DEFAULT '{"man": 0, "method": 0, "machine": 0, "material": 0}'::jsonb NOT NULL,
    variable_cost jsonb DEFAULT '{"man": 0, "method": 0, "machine": 0, "material": 0}'::jsonb NOT NULL
);


--
-- Name: tasks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tasks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tasks_id_seq OWNED BY public.tasks.id;


--
-- Name: unit_cost_assumption_rows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.unit_cost_assumption_rows (
    id bigint NOT NULL,
    unit_cost_assumption_id bigint NOT NULL,
    sort_order smallint NOT NULL,
    source_page smallint NOT NULL,
    row_type character varying(255) NOT NULL,
    section_code character varying(255),
    category character varying(255),
    cost_type character varying(255),
    component text,
    plan_quantity numeric(12,3),
    actual_quantity numeric(12,3),
    description text,
    unit text,
    base_price numeric(20,2),
    plan_daily_cost numeric(20,2),
    plan_hourly_cost numeric(20,2),
    actual_daily_cost numeric(20,2),
    actual_hourly_cost numeric(20,2),
    plan_value numeric(20,2),
    actual_value numeric(20,2),
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: unit_cost_assumption_rows_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.unit_cost_assumption_rows_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: unit_cost_assumption_rows_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.unit_cost_assumption_rows_id_seq OWNED BY public.unit_cost_assumption_rows.id;


--
-- Name: unit_cost_assumptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.unit_cost_assumptions (
    id bigint NOT NULL,
    code character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    assumption_date date NOT NULL,
    days_per_year smallint NOT NULL,
    days_per_month smallint NOT NULL,
    work_hours_per_day numeric(5,2) NOT NULL,
    source_sheet character varying(255) NOT NULL,
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    user_id bigint
);


--
-- Name: unit_cost_assumptions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.unit_cost_assumptions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: unit_cost_assumptions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.unit_cost_assumptions_id_seq OWNED BY public.unit_cost_assumptions.id;


--
-- Name: user_regional_assignments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_regional_assignments (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    scope_level character varying(32) NOT NULL,
    provinsi character varying(255) NOT NULL,
    kota_kabupaten character varying(255),
    kecamatan character varying(255),
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone
);


--
-- Name: user_regional_assignments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_regional_assignments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_regional_assignments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_regional_assignments_id_seq OWNED BY public.user_regional_assignments.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    email_verified_at timestamp(0) without time zone,
    password character varying(255) NOT NULL,
    remember_token character varying(100),
    created_at timestamp(0) without time zone,
    updated_at timestamp(0) without time zone,
    two_factor_secret text,
    two_factor_recovery_codes text,
    two_factor_confirmed_at timestamp(0) without time zone,
    role_id bigint,
    username character varying(255),
    sdm_kdkmp_entry_id bigint,
    has_completed_onboarding boolean DEFAULT false NOT NULL,
    manager_sk_document jsonb
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: business_process_steps id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_process_steps ALTER COLUMN id SET DEFAULT nextval('public.business_process_steps_id_seq'::regclass);


--
-- Name: business_processes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_processes ALTER COLUMN id SET DEFAULT nextval('public.business_processes_id_seq'::regclass);


--
-- Name: ebitda_values id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitda_values ALTER COLUMN id SET DEFAULT nextval('public.ebitda_values_id_seq'::regclass);


--
-- Name: ebitdamax_kdkmp id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitdamax_kdkmp ALTER COLUMN id SET DEFAULT nextval('public.ebitdamax_kdkmp_id_seq'::regclass);


--
-- Name: excel_imports id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.excel_imports ALTER COLUMN id SET DEFAULT nextval('public.excel_imports_id_seq'::regclass);


--
-- Name: failed_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.failed_jobs ALTER COLUMN id SET DEFAULT nextval('public.failed_jobs_id_seq'::regclass);


--
-- Name: import_error_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.import_error_logs ALTER COLUMN id SET DEFAULT nextval('public.import_error_logs_id_seq'::regclass);


--
-- Name: jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs ALTER COLUMN id SET DEFAULT nextval('public.jobs_id_seq'::regclass);


--
-- Name: koperasi_sarpras_status_points id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.koperasi_sarpras_status_points ALTER COLUMN id SET DEFAULT nextval('public.koperasi_sarpras_status_points_id_seq'::regclass);


--
-- Name: meeting_minute_attachments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_attachments ALTER COLUMN id SET DEFAULT nextval('public.meeting_minute_attachments_id_seq'::regclass);


--
-- Name: meeting_minute_item_status_histories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_item_status_histories ALTER COLUMN id SET DEFAULT nextval('public.meeting_minute_item_status_histories_id_seq'::regclass);


--
-- Name: meeting_minute_items id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_items ALTER COLUMN id SET DEFAULT nextval('public.meeting_minute_items_id_seq'::regclass);


--
-- Name: meeting_minutes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minutes ALTER COLUMN id SET DEFAULT nextval('public.meeting_minutes_id_seq'::regclass);


--
-- Name: migrations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.migrations ALTER COLUMN id SET DEFAULT nextval('public.migrations_id_seq'::regclass);


--
-- Name: organization_calculations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_calculations ALTER COLUMN id SET DEFAULT nextval('public.organization_calculations_id_seq'::regclass);


--
-- Name: organization_profiles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles ALTER COLUMN id SET DEFAULT nextval('public.organization_profiles_id_seq'::regclass);


--
-- Name: organizations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations ALTER COLUMN id SET DEFAULT nextval('public.organizations_id_seq'::regclass);


--
-- Name: passkeys id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.passkeys ALTER COLUMN id SET DEFAULT nextval('public.passkeys_id_seq'::regclass);


--
-- Name: plan_ebitda_matrices id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrices ALTER COLUMN id SET DEFAULT nextval('public.plan_ebitda_matrices_id_seq'::regclass);


--
-- Name: plan_ebitda_matrix_processes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_processes ALTER COLUMN id SET DEFAULT nextval('public.plan_ebitda_matrix_processes_id_seq'::regclass);


--
-- Name: plan_ebitda_matrix_rows id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_rows ALTER COLUMN id SET DEFAULT nextval('public.plan_ebitda_matrix_rows_id_seq'::regclass);


--
-- Name: revenue_plan_rows id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plan_rows ALTER COLUMN id SET DEFAULT nextval('public.revenue_plan_rows_id_seq'::regclass);


--
-- Name: revenue_plans id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plans ALTER COLUMN id SET DEFAULT nextval('public.revenue_plans_id_seq'::regclass);


--
-- Name: roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles ALTER COLUMN id SET DEFAULT nextval('public.roles_id_seq'::regclass);


--
-- Name: sdm_kdkmp_entries id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sdm_kdkmp_entries ALTER COLUMN id SET DEFAULT nextval('public.sdm_kdkmp_entries_id_seq'::regclass);


--
-- Name: task_additional_fields id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_additional_fields ALTER COLUMN id SET DEFAULT nextval('public.task_additional_fields_id_seq'::regclass);


--
-- Name: task_categories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_categories ALTER COLUMN id SET DEFAULT nextval('public.task_categories_id_seq'::regclass);


--
-- Name: task_report_values id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_report_values ALTER COLUMN id SET DEFAULT nextval('public.task_report_values_id_seq'::regclass);


--
-- Name: task_reports id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_reports ALTER COLUMN id SET DEFAULT nextval('public.task_reports_id_seq'::regclass);


--
-- Name: task_roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_roles ALTER COLUMN id SET DEFAULT nextval('public.task_roles_id_seq'::regclass);


--
-- Name: tasks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks ALTER COLUMN id SET DEFAULT nextval('public.tasks_id_seq'::regclass);


--
-- Name: unit_cost_assumption_rows id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumption_rows ALTER COLUMN id SET DEFAULT nextval('public.unit_cost_assumption_rows_id_seq'::regclass);


--
-- Name: unit_cost_assumptions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumptions ALTER COLUMN id SET DEFAULT nextval('public.unit_cost_assumptions_id_seq'::regclass);


--
-- Name: user_regional_assignments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_regional_assignments ALTER COLUMN id SET DEFAULT nextval('public.user_regional_assignments_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: business_process_steps business_process_steps_business_process_id_sequence_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_process_steps
    ADD CONSTRAINT business_process_steps_business_process_id_sequence_unique UNIQUE (business_process_id, sequence);


--
-- Name: business_process_steps business_process_steps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_process_steps
    ADD CONSTRAINT business_process_steps_pkey PRIMARY KEY (id);


--
-- Name: business_processes business_processes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_processes
    ADD CONSTRAINT business_processes_pkey PRIMARY KEY (id);


--
-- Name: business_processes business_processes_user_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_processes
    ADD CONSTRAINT business_processes_user_code_unique UNIQUE (user_id, code);


--
-- Name: cache_locks cache_locks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cache_locks
    ADD CONSTRAINT cache_locks_pkey PRIMARY KEY (key);


--
-- Name: cache cache_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cache
    ADD CONSTRAINT cache_pkey PRIMARY KEY (key);


--
-- Name: ebitda_values ebitda_values_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitda_values
    ADD CONSTRAINT ebitda_values_pkey PRIMARY KEY (id);


--
-- Name: ebitda_values ebitda_values_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitda_values
    ADD CONSTRAINT ebitda_values_unique UNIQUE (organization_id, year, period_date, scenario);


--
-- Name: ebitdamax_kdkmp ebitdamax_kdkmp_entry_date_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitdamax_kdkmp
    ADD CONSTRAINT ebitdamax_kdkmp_entry_date_unique UNIQUE (sdm_kdkmp_entry_id, report_date);


--
-- Name: ebitdamax_kdkmp ebitdamax_kdkmp_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitdamax_kdkmp
    ADD CONSTRAINT ebitdamax_kdkmp_pkey PRIMARY KEY (id);


--
-- Name: excel_imports excel_imports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.excel_imports
    ADD CONSTRAINT excel_imports_pkey PRIMARY KEY (id);


--
-- Name: failed_jobs failed_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.failed_jobs
    ADD CONSTRAINT failed_jobs_pkey PRIMARY KEY (id);


--
-- Name: failed_jobs failed_jobs_uuid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.failed_jobs
    ADD CONSTRAINT failed_jobs_uuid_unique UNIQUE (uuid);


--
-- Name: import_error_logs import_error_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.import_error_logs
    ADD CONSTRAINT import_error_logs_pkey PRIMARY KEY (id);


--
-- Name: job_batches job_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_batches
    ADD CONSTRAINT job_batches_pkey PRIMARY KEY (id);


--
-- Name: jobs jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);


--
-- Name: koperasi_sarpras_status_points koperasi_point_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.koperasi_sarpras_status_points
    ADD CONSTRAINT koperasi_point_unique UNIQUE (nik, lat, lng);


--
-- Name: koperasi_sarpras_status_points koperasi_sarpras_status_points_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.koperasi_sarpras_status_points
    ADD CONSTRAINT koperasi_sarpras_status_points_pkey PRIMARY KEY (id);


--
-- Name: meeting_minute_attachments meeting_minute_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_attachments
    ADD CONSTRAINT meeting_minute_attachments_pkey PRIMARY KEY (id);


--
-- Name: meeting_minute_item_status_histories meeting_minute_item_status_histories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_item_status_histories
    ADD CONSTRAINT meeting_minute_item_status_histories_pkey PRIMARY KEY (id);


--
-- Name: meeting_minute_items meeting_minute_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_items
    ADD CONSTRAINT meeting_minute_items_pkey PRIMARY KEY (id);


--
-- Name: meeting_minutes meeting_minutes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minutes
    ADD CONSTRAINT meeting_minutes_pkey PRIMARY KEY (id);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: organization_calculations organization_calculations_organization_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_calculations
    ADD CONSTRAINT organization_calculations_organization_id_unique UNIQUE (organization_id);


--
-- Name: organization_calculations organization_calculations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_calculations
    ADD CONSTRAINT organization_calculations_pkey PRIMARY KEY (id);


--
-- Name: organization_profiles organization_profiles_organization_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles
    ADD CONSTRAINT organization_profiles_organization_id_unique UNIQUE (organization_id);


--
-- Name: organization_profiles organization_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles
    ADD CONSTRAINT organization_profiles_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_code_unique UNIQUE (code);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_slug_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_slug_unique UNIQUE (slug);


--
-- Name: passkeys passkeys_credential_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.passkeys
    ADD CONSTRAINT passkeys_credential_id_unique UNIQUE (credential_id);


--
-- Name: passkeys passkeys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.passkeys
    ADD CONSTRAINT passkeys_pkey PRIMARY KEY (id);


--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (email);


--
-- Name: plan_ebitda_matrices plan_ebitda_matrices_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrices
    ADD CONSTRAINT plan_ebitda_matrices_pkey PRIMARY KEY (id);


--
-- Name: plan_ebitda_matrices plan_ebitda_matrices_user_id_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrices
    ADD CONSTRAINT plan_ebitda_matrices_user_id_code_unique UNIQUE (user_id, code);


--
-- Name: plan_ebitda_matrix_processes plan_ebitda_matrix_processes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_processes
    ADD CONSTRAINT plan_ebitda_matrix_processes_pkey PRIMARY KEY (id);


--
-- Name: plan_ebitda_matrix_processes plan_ebitda_matrix_processes_plan_ebitda_matrix_id_sequence_uni; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_processes
    ADD CONSTRAINT plan_ebitda_matrix_processes_plan_ebitda_matrix_id_sequence_uni UNIQUE (plan_ebitda_matrix_id, sequence);


--
-- Name: plan_ebitda_matrix_rows plan_ebitda_matrix_rows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_rows
    ADD CONSTRAINT plan_ebitda_matrix_rows_pkey PRIMARY KEY (id);


--
-- Name: plan_ebitda_matrix_rows plan_ebitda_matrix_rows_plan_ebitda_matrix_id_sort_order_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_rows
    ADD CONSTRAINT plan_ebitda_matrix_rows_plan_ebitda_matrix_id_sort_order_unique UNIQUE (plan_ebitda_matrix_id, sort_order);


--
-- Name: revenue_plan_rows revenue_plan_rows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plan_rows
    ADD CONSTRAINT revenue_plan_rows_pkey PRIMARY KEY (id);


--
-- Name: revenue_plan_rows revenue_plan_rows_revenue_plan_id_sort_order_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plan_rows
    ADD CONSTRAINT revenue_plan_rows_revenue_plan_id_sort_order_unique UNIQUE (revenue_plan_id, sort_order);


--
-- Name: revenue_plans revenue_plans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plans
    ADD CONSTRAINT revenue_plans_pkey PRIMARY KEY (id);


--
-- Name: revenue_plans revenue_plans_user_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plans
    ADD CONSTRAINT revenue_plans_user_code_unique UNIQUE (user_id, code);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: roles roles_slug_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_slug_unique UNIQUE (slug);


--
-- Name: roles roles_uuid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_uuid_unique UNIQUE (uuid);


--
-- Name: sdm_kdkmp_entries sdm_kdkmp_entries_nik_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sdm_kdkmp_entries
    ADD CONSTRAINT sdm_kdkmp_entries_nik_unique UNIQUE (nik);


--
-- Name: sdm_kdkmp_entries sdm_kdkmp_entries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sdm_kdkmp_entries
    ADD CONSTRAINT sdm_kdkmp_entries_pkey PRIMARY KEY (id);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: task_additional_fields task_additional_fields_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_additional_fields
    ADD CONSTRAINT task_additional_fields_pkey PRIMARY KEY (id);


--
-- Name: task_additional_fields task_additional_fields_task_id_field_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_additional_fields
    ADD CONSTRAINT task_additional_fields_task_id_field_name_unique UNIQUE (task_id, field_name);


--
-- Name: task_additional_fields task_additional_fields_uuid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_additional_fields
    ADD CONSTRAINT task_additional_fields_uuid_unique UNIQUE (uuid);


--
-- Name: task_categories task_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_categories
    ADD CONSTRAINT task_categories_pkey PRIMARY KEY (id);


--
-- Name: task_categories task_categories_slug_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_categories
    ADD CONSTRAINT task_categories_slug_unique UNIQUE (slug);


--
-- Name: task_categories task_categories_uuid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_categories
    ADD CONSTRAINT task_categories_uuid_unique UNIQUE (uuid);


--
-- Name: task_report_values task_report_values_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_report_values
    ADD CONSTRAINT task_report_values_pkey PRIMARY KEY (id);


--
-- Name: task_report_values task_report_values_task_report_id_task_additional_field_id_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_report_values
    ADD CONSTRAINT task_report_values_task_report_id_task_additional_field_id_uniq UNIQUE (task_report_id, task_additional_field_id);


--
-- Name: task_report_values task_report_values_uuid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_report_values
    ADD CONSTRAINT task_report_values_uuid_unique UNIQUE (uuid);


--
-- Name: task_reports task_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_reports
    ADD CONSTRAINT task_reports_pkey PRIMARY KEY (id);


--
-- Name: task_reports task_reports_uuid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_reports
    ADD CONSTRAINT task_reports_uuid_unique UNIQUE (uuid);


--
-- Name: task_roles task_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_roles
    ADD CONSTRAINT task_roles_pkey PRIMARY KEY (id);


--
-- Name: task_roles task_roles_task_id_role_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_roles
    ADD CONSTRAINT task_roles_task_id_role_id_unique UNIQUE (task_id, role_id);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);


--
-- Name: tasks tasks_sort_order_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_sort_order_unique UNIQUE (sort_order);


--
-- Name: tasks tasks_uuid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_uuid_unique UNIQUE (uuid);


--
-- Name: unit_cost_assumption_rows unit_cost_assumption_rows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumption_rows
    ADD CONSTRAINT unit_cost_assumption_rows_pkey PRIMARY KEY (id);


--
-- Name: unit_cost_assumption_rows unit_cost_assumption_rows_unit_cost_assumption_id_sort_order_un; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumption_rows
    ADD CONSTRAINT unit_cost_assumption_rows_unit_cost_assumption_id_sort_order_un UNIQUE (unit_cost_assumption_id, sort_order);


--
-- Name: unit_cost_assumptions unit_cost_assumptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumptions
    ADD CONSTRAINT unit_cost_assumptions_pkey PRIMARY KEY (id);


--
-- Name: unit_cost_assumptions unit_cost_assumptions_user_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumptions
    ADD CONSTRAINT unit_cost_assumptions_user_code_unique UNIQUE (user_id, code);


--
-- Name: user_regional_assignments user_regional_assignments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_regional_assignments
    ADD CONSTRAINT user_regional_assignments_pkey PRIMARY KEY (id);


--
-- Name: users users_email_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_unique UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_sdm_kdkmp_entry_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_sdm_kdkmp_entry_id_unique UNIQUE (sdm_kdkmp_entry_id);


--
-- Name: users users_username_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_unique UNIQUE (username);


--
-- Name: business_process_steps_business_process_id_process_group_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX business_process_steps_business_process_id_process_group_index ON public.business_process_steps USING btree (business_process_id, process_group);


--
-- Name: cache_expiration_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cache_expiration_index ON public.cache USING btree (expiration);


--
-- Name: cache_locks_expiration_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cache_locks_expiration_index ON public.cache_locks USING btree (expiration);


--
-- Name: ebitda_values_year_scenario_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ebitda_values_year_scenario_index ON public.ebitda_values USING btree (year, scenario);


--
-- Name: ebitdamax_kdkmp_plan_revenue_requires_review_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ebitdamax_kdkmp_plan_revenue_requires_review_index ON public.ebitdamax_kdkmp USING btree (plan_revenue_requires_review);


--
-- Name: ebitdamax_kdkmp_report_date_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ebitdamax_kdkmp_report_date_index ON public.ebitdamax_kdkmp USING btree (report_date);


--
-- Name: failed_jobs_connection_queue_failed_at_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX failed_jobs_connection_queue_failed_at_index ON public.failed_jobs USING btree (connection, queue, failed_at);


--
-- Name: jobs_queue_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_queue_index ON public.jobs USING btree (queue);


--
-- Name: koperasi_sarpras_status_points_nik_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX koperasi_sarpras_status_points_nik_index ON public.koperasi_sarpras_status_points USING btree (nik);


--
-- Name: meeting_item_status_history_timeline; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX meeting_item_status_history_timeline ON public.meeting_minute_item_status_histories USING btree (meeting_minute_item_id, created_at);


--
-- Name: meeting_minute_attachments_meeting_minute_id_created_at_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX meeting_minute_attachments_meeting_minute_id_created_at_index ON public.meeting_minute_attachments USING btree (meeting_minute_id, created_at);


--
-- Name: notifications_notifiable_type_notifiable_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_notifiable_type_notifiable_id_index ON public.notifications USING btree (notifiable_type, notifiable_id);


--
-- Name: organization_calculations_source_sheet_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX organization_calculations_source_sheet_index ON public.organization_calculations USING btree (source_sheet);


--
-- Name: organization_profiles_source_sheet_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX organization_profiles_source_sheet_index ON public.organization_profiles USING btree (source_sheet);


--
-- Name: organizations_code_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX organizations_code_index ON public.organizations USING btree (code);


--
-- Name: organizations_parent_id_sort_order_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX organizations_parent_id_sort_order_index ON public.organizations USING btree (parent_id, sort_order);


--
-- Name: organizations_path_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX organizations_path_index ON public.organizations USING btree (path);


--
-- Name: passkeys_user_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX passkeys_user_id_index ON public.passkeys USING btree (user_id);


--
-- Name: plan_ebitda_matrices_user_id_created_at_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX plan_ebitda_matrices_user_id_created_at_index ON public.plan_ebitda_matrices USING btree (user_id, created_at);


--
-- Name: plan_ebitda_matrix_rows_plan_ebitda_matrix_id_section_code_inde; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX plan_ebitda_matrix_rows_plan_ebitda_matrix_id_section_code_inde ON public.plan_ebitda_matrix_rows USING btree (plan_ebitda_matrix_id, section_code);


--
-- Name: revenue_plan_rows_revenue_plan_id_row_type_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX revenue_plan_rows_revenue_plan_id_row_type_index ON public.revenue_plan_rows USING btree (revenue_plan_id, row_type);


--
-- Name: revenue_plans_user_id_plan_date_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX revenue_plans_user_id_plan_date_index ON public.revenue_plans USING btree (user_id, plan_date);


--
-- Name: roles_domain_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX roles_domain_index ON public.roles USING btree (domain);


--
-- Name: roles_level_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX roles_level_index ON public.roles USING btree (level);


--
-- Name: roles_name_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX roles_name_index ON public.roles USING btree (name);


--
-- Name: sdm_kdkmp_entries_region_hierarchy_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sdm_kdkmp_entries_region_hierarchy_index ON public.sdm_kdkmp_entries USING btree (provinsi, kota_kabupaten, kecamatan);


--
-- Name: sessions_last_activity_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_last_activity_index ON public.sessions USING btree (last_activity);


--
-- Name: sessions_user_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_user_id_index ON public.sessions USING btree (user_id);


--
-- Name: task_additional_fields_input_type_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX task_additional_fields_input_type_index ON public.task_additional_fields USING btree (input_type);


--
-- Name: task_additional_fields_show_when_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX task_additional_fields_show_when_index ON public.task_additional_fields USING btree (show_when);


--
-- Name: task_categories_name_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX task_categories_name_index ON public.task_categories USING btree (name);


--
-- Name: task_reports_task_id_user_id_period_key_status_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX task_reports_task_id_user_id_period_key_status_index ON public.task_reports USING btree (task_id, user_id, period_key, status);


--
-- Name: task_reports_task_id_user_id_status_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX task_reports_task_id_user_id_status_index ON public.task_reports USING btree (task_id, user_id, status);


--
-- Name: task_roles_role_id_task_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX task_roles_role_id_task_id_index ON public.task_roles USING btree (role_id, task_id);


--
-- Name: tasks_is_active_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX tasks_is_active_index ON public.tasks USING btree (is_active);


--
-- Name: tasks_name_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX tasks_name_index ON public.tasks USING btree (name);


--
-- Name: tasks_period_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX tasks_period_index ON public.tasks USING btree (period);


--
-- Name: unit_cost_assumption_rows_unit_cost_assumption_id_row_type_inde; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX unit_cost_assumption_rows_unit_cost_assumption_id_row_type_inde ON public.unit_cost_assumption_rows USING btree (unit_cost_assumption_id, row_type);


--
-- Name: unit_cost_assumption_rows_unit_cost_assumption_id_section_code_; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX unit_cost_assumption_rows_unit_cost_assumption_id_section_code_ ON public.unit_cost_assumption_rows USING btree (unit_cost_assumption_id, section_code);


--
-- Name: user_regional_assignments_provinsi_kota_kabupaten_kecamatan_ind; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_regional_assignments_provinsi_kota_kabupaten_kecamatan_ind ON public.user_regional_assignments USING btree (provinsi, kota_kabupaten, kecamatan);


--
-- Name: user_regional_assignments_user_id_scope_level_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_regional_assignments_user_id_scope_level_index ON public.user_regional_assignments USING btree (user_id, scope_level);


--
-- Name: users_name_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX users_name_index ON public.users USING btree (name);


--
-- Name: business_process_steps business_process_steps_business_process_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_process_steps
    ADD CONSTRAINT business_process_steps_business_process_id_foreign FOREIGN KEY (business_process_id) REFERENCES public.business_processes(id) ON DELETE CASCADE;


--
-- Name: business_processes business_processes_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.business_processes
    ADD CONSTRAINT business_processes_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: ebitda_values ebitda_values_excel_import_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitda_values
    ADD CONSTRAINT ebitda_values_excel_import_id_foreign FOREIGN KEY (excel_import_id) REFERENCES public.excel_imports(id) ON DELETE SET NULL;


--
-- Name: ebitda_values ebitda_values_organization_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitda_values
    ADD CONSTRAINT ebitda_values_organization_id_foreign FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: ebitdamax_kdkmp ebitdamax_kdkmp_created_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitdamax_kdkmp
    ADD CONSTRAINT ebitdamax_kdkmp_created_by_foreign FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: ebitdamax_kdkmp ebitdamax_kdkmp_sdm_kdkmp_entry_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitdamax_kdkmp
    ADD CONSTRAINT ebitdamax_kdkmp_sdm_kdkmp_entry_id_foreign FOREIGN KEY (sdm_kdkmp_entry_id) REFERENCES public.sdm_kdkmp_entries(id) ON DELETE RESTRICT;


--
-- Name: ebitdamax_kdkmp ebitdamax_kdkmp_updated_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ebitdamax_kdkmp
    ADD CONSTRAINT ebitdamax_kdkmp_updated_by_foreign FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: excel_imports excel_imports_created_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.excel_imports
    ADD CONSTRAINT excel_imports_created_by_foreign FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: import_error_logs import_error_logs_excel_import_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.import_error_logs
    ADD CONSTRAINT import_error_logs_excel_import_id_foreign FOREIGN KEY (excel_import_id) REFERENCES public.excel_imports(id) ON DELETE CASCADE;


--
-- Name: meeting_minute_attachments meeting_minute_attachments_meeting_minute_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_attachments
    ADD CONSTRAINT meeting_minute_attachments_meeting_minute_id_foreign FOREIGN KEY (meeting_minute_id) REFERENCES public.meeting_minutes(id) ON DELETE CASCADE;


--
-- Name: meeting_minute_attachments meeting_minute_attachments_uploaded_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_attachments
    ADD CONSTRAINT meeting_minute_attachments_uploaded_by_foreign FOREIGN KEY (uploaded_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: meeting_minute_item_status_histories meeting_minute_item_status_histories_changed_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_item_status_histories
    ADD CONSTRAINT meeting_minute_item_status_histories_changed_by_foreign FOREIGN KEY (changed_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: meeting_minute_item_status_histories meeting_minute_item_status_histories_meeting_minute_item_id_for; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_item_status_histories
    ADD CONSTRAINT meeting_minute_item_status_histories_meeting_minute_item_id_for FOREIGN KEY (meeting_minute_item_id) REFERENCES public.meeting_minute_items(id) ON DELETE CASCADE;


--
-- Name: meeting_minute_items meeting_minute_items_meeting_minute_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minute_items
    ADD CONSTRAINT meeting_minute_items_meeting_minute_id_foreign FOREIGN KEY (meeting_minute_id) REFERENCES public.meeting_minutes(id) ON DELETE CASCADE;


--
-- Name: meeting_minutes meeting_minutes_created_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minutes
    ADD CONSTRAINT meeting_minutes_created_by_foreign FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: meeting_minutes meeting_minutes_updated_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.meeting_minutes
    ADD CONSTRAINT meeting_minutes_updated_by_foreign FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: organization_calculations organization_calculations_organization_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_calculations
    ADD CONSTRAINT organization_calculations_organization_id_foreign FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: organization_profiles organization_profiles_excel_import_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles
    ADD CONSTRAINT organization_profiles_excel_import_id_foreign FOREIGN KEY (excel_import_id) REFERENCES public.excel_imports(id) ON DELETE SET NULL;


--
-- Name: organization_profiles organization_profiles_organization_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_profiles
    ADD CONSTRAINT organization_profiles_organization_id_foreign FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: organizations organizations_parent_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_parent_id_foreign FOREIGN KEY (parent_id) REFERENCES public.organizations(id) ON DELETE SET NULL;


--
-- Name: passkeys passkeys_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.passkeys
    ADD CONSTRAINT passkeys_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: plan_ebitda_matrices plan_ebitda_matrices_business_process_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrices
    ADD CONSTRAINT plan_ebitda_matrices_business_process_id_foreign FOREIGN KEY (business_process_id) REFERENCES public.business_processes(id) ON DELETE SET NULL;


--
-- Name: plan_ebitda_matrices plan_ebitda_matrices_revenue_plan_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrices
    ADD CONSTRAINT plan_ebitda_matrices_revenue_plan_id_foreign FOREIGN KEY (revenue_plan_id) REFERENCES public.revenue_plans(id) ON DELETE SET NULL;


--
-- Name: plan_ebitda_matrices plan_ebitda_matrices_unit_cost_assumption_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrices
    ADD CONSTRAINT plan_ebitda_matrices_unit_cost_assumption_id_foreign FOREIGN KEY (unit_cost_assumption_id) REFERENCES public.unit_cost_assumptions(id) ON DELETE SET NULL;


--
-- Name: plan_ebitda_matrices plan_ebitda_matrices_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrices
    ADD CONSTRAINT plan_ebitda_matrices_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: plan_ebitda_matrix_processes plan_ebitda_matrix_processes_business_process_step_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_processes
    ADD CONSTRAINT plan_ebitda_matrix_processes_business_process_step_id_foreign FOREIGN KEY (business_process_step_id) REFERENCES public.business_process_steps(id) ON DELETE SET NULL;


--
-- Name: plan_ebitda_matrix_processes plan_ebitda_matrix_processes_plan_ebitda_matrix_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_processes
    ADD CONSTRAINT plan_ebitda_matrix_processes_plan_ebitda_matrix_id_foreign FOREIGN KEY (plan_ebitda_matrix_id) REFERENCES public.plan_ebitda_matrices(id) ON DELETE CASCADE;


--
-- Name: plan_ebitda_matrix_rows plan_ebitda_matrix_rows_plan_ebitda_matrix_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_rows
    ADD CONSTRAINT plan_ebitda_matrix_rows_plan_ebitda_matrix_id_foreign FOREIGN KEY (plan_ebitda_matrix_id) REFERENCES public.plan_ebitda_matrices(id) ON DELETE CASCADE;


--
-- Name: plan_ebitda_matrix_rows plan_ebitda_matrix_rows_unit_cost_assumption_row_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_ebitda_matrix_rows
    ADD CONSTRAINT plan_ebitda_matrix_rows_unit_cost_assumption_row_id_foreign FOREIGN KEY (unit_cost_assumption_row_id) REFERENCES public.unit_cost_assumption_rows(id) ON DELETE SET NULL;


--
-- Name: revenue_plan_rows revenue_plan_rows_revenue_plan_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plan_rows
    ADD CONSTRAINT revenue_plan_rows_revenue_plan_id_foreign FOREIGN KEY (revenue_plan_id) REFERENCES public.revenue_plans(id) ON DELETE CASCADE;


--
-- Name: revenue_plans revenue_plans_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue_plans
    ADD CONSTRAINT revenue_plans_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: sdm_kdkmp_entries sdm_kdkmp_entries_created_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sdm_kdkmp_entries
    ADD CONSTRAINT sdm_kdkmp_entries_created_by_foreign FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: sdm_kdkmp_entries sdm_kdkmp_entries_updated_by_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sdm_kdkmp_entries
    ADD CONSTRAINT sdm_kdkmp_entries_updated_by_foreign FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: task_additional_fields task_additional_fields_task_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_additional_fields
    ADD CONSTRAINT task_additional_fields_task_id_foreign FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: task_report_values task_report_values_task_additional_field_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_report_values
    ADD CONSTRAINT task_report_values_task_additional_field_id_foreign FOREIGN KEY (task_additional_field_id) REFERENCES public.task_additional_fields(id) ON DELETE CASCADE;


--
-- Name: task_report_values task_report_values_task_report_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_report_values
    ADD CONSTRAINT task_report_values_task_report_id_foreign FOREIGN KEY (task_report_id) REFERENCES public.task_reports(id) ON DELETE CASCADE;


--
-- Name: task_reports task_reports_task_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_reports
    ADD CONSTRAINT task_reports_task_id_foreign FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: task_reports task_reports_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_reports
    ADD CONSTRAINT task_reports_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: task_roles task_roles_role_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_roles
    ADD CONSTRAINT task_roles_role_id_foreign FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE RESTRICT;


--
-- Name: task_roles task_roles_task_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_roles
    ADD CONSTRAINT task_roles_task_id_foreign FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: tasks tasks_task_category_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_task_category_id_foreign FOREIGN KEY (task_category_id) REFERENCES public.task_categories(id) ON DELETE RESTRICT;


--
-- Name: unit_cost_assumption_rows unit_cost_assumption_rows_unit_cost_assumption_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumption_rows
    ADD CONSTRAINT unit_cost_assumption_rows_unit_cost_assumption_id_foreign FOREIGN KEY (unit_cost_assumption_id) REFERENCES public.unit_cost_assumptions(id) ON DELETE CASCADE;


--
-- Name: unit_cost_assumptions unit_cost_assumptions_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unit_cost_assumptions
    ADD CONSTRAINT unit_cost_assumptions_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: user_regional_assignments user_regional_assignments_user_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_regional_assignments
    ADD CONSTRAINT user_regional_assignments_user_id_foreign FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_role_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_role_id_foreign FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE SET NULL;


--
-- Name: users users_sdm_kdkmp_entry_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_sdm_kdkmp_entry_id_foreign FOREIGN KEY (sdm_kdkmp_entry_id) REFERENCES public.sdm_kdkmp_entries(id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--



-- +goose Down
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO ebitdamax;
GRANT ALL ON SCHEMA public TO public;
