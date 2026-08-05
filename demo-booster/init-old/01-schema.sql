--
-- PostgreSQL database dump
--


-- Dumped from database version 16.14 (Debian 16.14-1.pgdg13+1)
-- Dumped by pg_dump version 16.14 (Debian 16.14-1.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: sentiment_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.sentiment_type AS ENUM (
    'NEGATIVE',
    'NEUTRAL',
    'POSITIVE'
);


--
-- Name: n8n_trigger_function_0421e4c2_155e_4d22_b5f4_600e4845a5f1(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.n8n_trigger_function_0421e4c2_155e_4d22_b5f4_600e4845a5f1() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ begin perform pg_notify('n8n_channel_0421e4c2_155e_4d22_b5f4_600e4845a5f1', row_to_json(new)::text); return null; end; $$;


--
-- Name: n8n_trigger_function_25ebbf08_ec3e_4881_9b70_538cc6af5ab8(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.n8n_trigger_function_25ebbf08_ec3e_4881_9b70_538cc6af5ab8() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ begin perform pg_notify('n8n_channel_25ebbf08_ec3e_4881_9b70_538cc6af5ab8', row_to_json(new)::text); return null; end; $$;


--
-- Name: n8n_trigger_function_ae5e7d23_251d_4612_b4ea_6ca75de169f4(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.n8n_trigger_function_ae5e7d23_251d_4612_b4ea_6ca75de169f4() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ begin perform pg_notify('n8n_channel_ae5e7d23_251d_4612_b4ea_6ca75de169f4', row_to_json(new)::text); return null; end; $$;


--
-- Name: admin_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admin_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


SET default_table_access_method = heap;

--
-- Name: ai_analysis_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ai_analysis_log (
    id bigint NOT NULL,
    group_id bigint NOT NULL,
    store_id bigint NOT NULL,
    channel character varying(30) NOT NULL,
    channel_analysis_info_id bigint NOT NULL,
    status character varying(10) DEFAULT 'PENDING'::character varying NOT NULL,
    total_data_count integer DEFAULT 0 NOT NULL,
    complete_data_count integer DEFAULT 0 NOT NULL,
    started_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    ended_at timestamp with time zone,
    description character varying(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);


--
-- Name: TABLE ai_analysis_log; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.ai_analysis_log IS '분석 수행 로그';


--
-- Name: COLUMN ai_analysis_log.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.id IS 'id';


--
-- Name: COLUMN ai_analysis_log.group_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.group_id IS '그룹핑 id';


--
-- Name: COLUMN ai_analysis_log.store_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.store_id IS '분석 대상 store';


--
-- Name: COLUMN ai_analysis_log.channel; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.channel IS '분석 대상 channel';


--
-- Name: COLUMN ai_analysis_log.channel_analysis_info_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.channel_analysis_info_id IS 'ai 모델 유형 id';


--
-- Name: COLUMN ai_analysis_log.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.status IS '상태 (PENDING:분석중/COMPLETE:완료)';


--
-- Name: COLUMN ai_analysis_log.total_data_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.total_data_count IS '분석 데이터 총 건수';


--
-- Name: COLUMN ai_analysis_log.complete_data_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.complete_data_count IS '분석 데이터 총 건수';


--
-- Name: COLUMN ai_analysis_log.started_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.started_at IS '분석 시작일자';


--
-- Name: COLUMN ai_analysis_log.ended_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.ended_at IS '분석 종료일자';


--
-- Name: COLUMN ai_analysis_log.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.created_at IS '생성일자';


--
-- Name: COLUMN ai_analysis_log.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_analysis_log.updated_at IS '수정일자';


--
-- Name: ai_analysis_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ai_analysis_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ai_analysis_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ai_analysis_log_id_seq OWNED BY public.ai_analysis_log.id;


--
-- Name: ai_model_version; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ai_model_version (
    id bigint NOT NULL,
    model character varying(20) NOT NULL,
    version character varying(30) NOT NULL,
    usable boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: TABLE ai_model_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.ai_model_version IS 'ai 모델 버전 정보';


--
-- Name: COLUMN ai_model_version.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_model_version.id IS 'id';


--
-- Name: COLUMN ai_model_version.model; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_model_version.model IS 'ai 모델';


--
-- Name: COLUMN ai_model_version.version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_model_version.version IS 'ai 버전';


--
-- Name: COLUMN ai_model_version.usable; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_model_version.usable IS '사용 여부';


--
-- Name: COLUMN ai_model_version.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ai_model_version.created_at IS '생성일자';


--
-- Name: ai_model_version_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ai_model_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ai_model_version_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ai_model_version_id_seq OWNED BY public.ai_model_version.id;


--
-- Name: app_user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: app_user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_user (
    id bigint DEFAULT nextval('public.app_user_id_seq'::regclass) NOT NULL,
    email character varying(255) NOT NULL,
    app_user_name character varying(255) NOT NULL,
    phone character varying(255),
    status character varying(255) NOT NULL,
    kakao_id bigint,
    azure_id character varying(255),
    google_id character varying(255),
    naver_id character varying(255),
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    role character varying DEFAULT 'ROLE_USER'::character varying NOT NULL
);


--
-- Name: COLUMN app_user.email; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.app_user.email IS 'Todo.';


--
-- Name: COLUMN app_user.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.app_user.status IS 'Todo.';


--
-- Name: app_user_attribute; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_user_attribute (
    id bigint NOT NULL,
    app_user_id bigint NOT NULL,
    attribute character varying NOT NULL,
    value character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: app_user_attribute_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_user_attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: app_user_attribute_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.app_user_attribute_id_seq OWNED BY public.app_user_attribute.id;


--
-- Name: app_user_channel_account_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_user_channel_account_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: app_user_social; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_user_social (
    id bigint NOT NULL,
    app_user_id bigint NOT NULL,
    type character varying(10) NOT NULL,
    social_identifier character varying(200) NOT NULL,
    usable boolean DEFAULT true NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: app_user_social_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_user_social_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: app_user_social_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.app_user_social_id_seq OWNED BY public.app_user_social.id;


--
-- Name: app_user_store_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.app_user_store_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: app_user_store; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_user_store (
    id bigint DEFAULT nextval('public.app_user_store_id_seq'::regclass) NOT NULL,
    app_user_id bigint NOT NULL,
    store_id bigint NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    analysis_start_at timestamp with time zone,
    analysis_end_at timestamp with time zone,
    status character varying(255) NOT NULL,
    store_name character varying(255),
    store_detail character varying(255)
);


--
-- Name: COLUMN app_user_store.analysis_start_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.app_user_store.analysis_start_at IS '참고용';


--
-- Name: COLUMN app_user_store.analysis_end_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.app_user_store.analysis_end_at IS '참고용';


--
-- Name: COLUMN app_user_store.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.app_user_store.status IS '참고용';


--
-- Name: boost_answer_code; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_answer_code (
    name character varying(100) NOT NULL,
    code character varying(6) NOT NULL
);


--
-- Name: boost_answer_code_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_answer_code_seq
    START WITH 1
    INCREMENT BY 1
    MINVALUE 0
    MAXVALUE 2176782335
    CACHE 1;


--
-- Name: boost_answer_icon; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_answer_icon (
    id bigint NOT NULL,
    summary character varying(100),
    icon_url character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: boost_answer_icon_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_answer_icon_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_answer_icon_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_answer_icon_id_seq OWNED BY public.boost_answer_icon.id;


--
-- Name: boost_answer_map; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_answer_map (
    id bigint NOT NULL,
    code character varying(6) NOT NULL,
    store_id bigint NOT NULL,
    descriptions character varying(100),
    display_order integer DEFAULT 0 NOT NULL,
    icon_id bigint,
    boost_question_id bigint NOT NULL,
    type character varying(10) NOT NULL
);


--
-- Name: boost_answer_map_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_answer_map_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_answer_map_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_answer_map_id_seq OWNED BY public.boost_answer_map.id;


--
-- Name: boost_customer; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_customer (
    id character varying NOT NULL,
    customer_name character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: COLUMN boost_customer.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_customer.id IS 'phone_number';


--
-- Name: boost_customer_gift; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_customer_gift (
    customer_id character varying NOT NULL,
    post_id character varying NOT NULL,
    gift_id character varying NOT NULL,
    is_valid boolean DEFAULT false NOT NULL,
    is_used boolean DEFAULT false NOT NULL,
    usable boolean NOT NULL,
    expired_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    message_id character varying,
    id character varying NOT NULL,
    team_id character varying
);


--
-- Name: COLUMN boost_customer_gift.customer_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_customer_gift.customer_id IS '?? phone으로 해도 괜찮을까??';


--
-- Name: COLUMN boost_customer_gift.post_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_customer_gift.post_id IS 'UUID';


--
-- Name: boost_dental_image; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_dental_image (
    id bigint NOT NULL,
    dental_patient_id bigint NOT NULL,
    filename character varying(255),
    type integer,
    file_stream bytea,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: boost_dental_image_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_dental_image_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_dental_image_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_dental_image_id_seq OWNED BY public.boost_dental_image.id;


--
-- Name: boost_dental_patient_info; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_dental_patient_info (
    id bigint NOT NULL,
    hospital_id bigint,
    patient_id bigint,
    gender character varying(10),
    birth_date date,
    chart_number character varying(20),
    treatment_date date,
    patient_name character varying(20),
    reservation_notes character varying(200),
    checkin_notes character varying(200),
    treatment_content character varying(1000),
    doctor_name character varying(20),
    reservation_time timestamp without time zone
);


--
-- Name: boost_dental_patient_info_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_dental_patient_info_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_dental_patient_info_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_dental_patient_info_id_seq OWNED BY public.boost_dental_patient_info.id;


--
-- Name: boost_gift; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_gift (
    id character varying NOT NULL,
    name character varying,
    display_name character varying,
    is_valid boolean NOT NULL,
    provider character varying,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: COLUMN boost_gift.is_valid; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_gift.is_valid IS 'LOGIN, JOIN';


--
-- Name: boost_message; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_message (
    id character varying NOT NULL,
    team_id character varying NOT NULL,
    customer_id character varying NOT NULL,
    treatment character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    link_mobile character varying,
    link_pc character varying,
    link_expired_at timestamp with time zone,
    message_content character varying,
    link_used_at timestamp with time zone,
    customer_name character varying,
    message_template_id character varying
);


--
-- Name: COLUMN boost_message.customer_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_message.customer_id IS '?? phone으로 해도 괜찮을까??';


--
-- Name: boost_message_template; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_message_template (
    id character varying NOT NULL,
    template_name character varying,
    template_content character varying,
    plus_friend_id character varying,
    template_code character varying,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    button_type character varying,
    button_name character varying,
    link_mobile character varying,
    link_pc character varying,
    message_type character varying
);


--
-- Name: COLUMN boost_message_template.plus_friend_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_message_template.plus_friend_id IS 'LOGIN, JOIN';


--
-- Name: boost_post_similarity; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_post_similarity (
    id bigint NOT NULL,
    store_id bigint NOT NULL,
    post_id character varying(50) NOT NULL,
    enhanced_post_id character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);


--
-- Name: COLUMN boost_post_similarity.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_post_similarity.id IS 'id';


--
-- Name: COLUMN boost_post_similarity.store_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_post_similarity.store_id IS 'store id';


--
-- Name: COLUMN boost_post_similarity.post_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_post_similarity.post_id IS '수집 post id';


--
-- Name: COLUMN boost_post_similarity.enhanced_post_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_post_similarity.enhanced_post_id IS '리뷰부스터 ai 생성 post id';


--
-- Name: COLUMN boost_post_similarity.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_post_similarity.created_at IS '생성일자';


--
-- Name: COLUMN boost_post_similarity.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_post_similarity.updated_at IS '수정일자';


--
-- Name: boost_post_similarity_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_post_similarity_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_post_similarity_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_post_similarity_id_seq OWNED BY public.boost_post_similarity.id;


--
-- Name: boost_present; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_present (
    id bigint NOT NULL,
    store_id bigint NOT NULL,
    platform character varying(50) NOT NULL,
    present_token character varying(100) NOT NULL,
    item_name character varying(30) NOT NULL,
    item_price numeric(13,4) DEFAULT 0.0000 NOT NULL,
    status character varying(25) DEFAULT 'REGISTERED'::character varying NOT NULL,
    sending_count integer DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE boost_present; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.boost_present IS '리뷰부스터 선물 정보';


--
-- Name: COLUMN boost_present.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.id IS 'id';


--
-- Name: COLUMN boost_present.store_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.store_id IS 'store id';


--
-- Name: COLUMN boost_present.platform; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.platform IS 'platform (KAKAO)';


--
-- Name: COLUMN boost_present.present_token; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.present_token IS '선물 플랫폼 token 값';


--
-- Name: COLUMN boost_present.item_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.item_name IS '상품명';


--
-- Name: COLUMN boost_present.item_price; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.item_price IS '상품 가격';


--
-- Name: COLUMN boost_present.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.status IS '선물 상태 (REGISTERED:대기중/ACTIVE:활성호ㅓ/EXPIRED:발송기간만료/CLOSED:정지)';


--
-- Name: COLUMN boost_present.sending_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.sending_count IS '누적 발송 개수';


--
-- Name: COLUMN boost_present.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.created_at IS '생성일자';


--
-- Name: COLUMN boost_present.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present.updated_at IS '수정일자';


--
-- Name: boost_present_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_present_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_present_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_present_id_seq OWNED BY public.boost_present.id;


--
-- Name: boost_present_ledger; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_present_ledger (
    id bigint NOT NULL,
    store_id bigint NOT NULL,
    order_id bigint NOT NULL,
    item_name character varying(30) NOT NULL,
    item_count integer DEFAULT 0 NOT NULL,
    price numeric(13,4) DEFAULT 0.0000 NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE boost_present_ledger; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.boost_present_ledger IS '리뷰부스터 선물 주문 히스토리';


--
-- Name: COLUMN boost_present_ledger.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_ledger.id IS 'id';


--
-- Name: COLUMN boost_present_ledger.store_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_ledger.store_id IS 'store id';


--
-- Name: COLUMN boost_present_ledger.order_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_ledger.order_id IS 'order id';


--
-- Name: COLUMN boost_present_ledger.item_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_ledger.item_name IS '상품명';


--
-- Name: COLUMN boost_present_ledger.item_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_ledger.item_count IS '상품개수';


--
-- Name: COLUMN boost_present_ledger.price; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_ledger.price IS '가격';


--
-- Name: COLUMN boost_present_ledger.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_ledger.created_at IS '생성일자';


--
-- Name: boost_present_ledger_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_present_ledger_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_present_ledger_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_present_ledger_id_seq OWNED BY public.boost_present_ledger.id;


--
-- Name: boost_present_order; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_present_order (
    id bigint NOT NULL,
    uuid uuid NOT NULL,
    store_id bigint NOT NULL,
    receipt_id bigint,
    message_id character varying(50),
    boost_present_id bigint NOT NULL,
    receiver character varying(25) NOT NULL,
    result character varying(20) DEFAULT 'WAIT'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE boost_present_order; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.boost_present_order IS '리뷰부스터 선물 주문';


--
-- Name: COLUMN boost_present_order.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.id IS 'id';


--
-- Name: COLUMN boost_present_order.uuid; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.uuid IS 'uuid';


--
-- Name: COLUMN boost_present_order.store_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.store_id IS 'store id';


--
-- Name: COLUMN boost_present_order.receipt_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.receipt_id IS 'receipt id';


--
-- Name: COLUMN boost_present_order.message_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.message_id IS 'message id';


--
-- Name: COLUMN boost_present_order.boost_present_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.boost_present_id IS '리뷰부스터 선물 id';


--
-- Name: COLUMN boost_present_order.receiver; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.receiver IS '수신자 정보';


--
-- Name: COLUMN boost_present_order.result; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.result IS '발송 결과 (WAIT:발송요청/SUCCESS:성공/FAIL:실패)';


--
-- Name: COLUMN boost_present_order.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.created_at IS '생성일자';


--
-- Name: COLUMN boost_present_order.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order.updated_at IS '수정일자';


--
-- Name: boost_present_order_attribute; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_present_order_attribute (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    key character varying(25) NOT NULL,
    value character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE boost_present_order_attribute; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.boost_present_order_attribute IS '리뷰부스터 선물 주문 속성';


--
-- Name: COLUMN boost_present_order_attribute.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_attribute.id IS 'id';


--
-- Name: COLUMN boost_present_order_attribute.order_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_attribute.order_id IS '주문 id';


--
-- Name: COLUMN boost_present_order_attribute.key; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_attribute.key IS '속성 키';


--
-- Name: COLUMN boost_present_order_attribute.value; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_attribute.value IS '속성 값';


--
-- Name: COLUMN boost_present_order_attribute.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_attribute.created_at IS '생성일자';


--
-- Name: boost_present_order_attribute_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_present_order_attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_present_order_attribute_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_present_order_attribute_id_seq OWNED BY public.boost_present_order_attribute.id;


--
-- Name: boost_present_order_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_present_order_history (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    transaction_id character varying(50),
    result character varying(20) DEFAULT 'WAIT'::character varying NOT NULL,
    error_message character varying(255),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE boost_present_order_history; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.boost_present_order_history IS '리뷰부스터 선물 주문 히스토리';


--
-- Name: COLUMN boost_present_order_history.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_history.id IS 'id';


--
-- Name: COLUMN boost_present_order_history.order_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_history.order_id IS 'boost present order id';


--
-- Name: COLUMN boost_present_order_history.transaction_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_history.transaction_id IS 'transaction id';


--
-- Name: COLUMN boost_present_order_history.result; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_history.result IS '발송 결과 (WAIT:발송요청/SUCCESS:성공/FAIL:실패)';


--
-- Name: COLUMN boost_present_order_history.error_message; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_history.error_message IS '실패 메세지';


--
-- Name: COLUMN boost_present_order_history.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_history.created_at IS '생성일자';


--
-- Name: COLUMN boost_present_order_history.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_present_order_history.updated_at IS '수정일자';


--
-- Name: boost_present_order_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_present_order_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_present_order_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_present_order_history_id_seq OWNED BY public.boost_present_order_history.id;


--
-- Name: boost_present_order_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_present_order_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_present_order_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_present_order_id_seq OWNED BY public.boost_present_order.id;


--
-- Name: boost_question_code; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_question_code (
    name character varying(100) NOT NULL,
    code character varying(6) NOT NULL
);


--
-- Name: boost_question_code_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_question_code_seq
    START WITH 1
    INCREMENT BY 1
    MINVALUE 0
    MAXVALUE 2176782335
    CACHE 1;


--
-- Name: boost_question_map; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_question_map (
    id bigint NOT NULL,
    code character varying(6) NOT NULL,
    store_id bigint,
    type character varying(20),
    usable boolean DEFAULT true NOT NULL,
    placeholder character varying(255),
    display_order integer DEFAULT 0 NOT NULL,
    selectable_items integer DEFAULT 1 NOT NULL,
    sub_title character varying(255)
);


--
-- Name: boost_question_map_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_question_map_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_question_map_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_question_map_id_seq OWNED BY public.boost_question_map.id;


--
-- Name: boost_receipt_info; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_receipt_info (
    id bigint NOT NULL,
    store_id bigint NOT NULL,
    use_crawling boolean DEFAULT false NOT NULL,
    use_kakao_talk_business boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: COLUMN boost_receipt_info.use_crawling; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_info.use_crawling IS '영수증 크롤링 수행 여부';


--
-- Name: COLUMN boost_receipt_info.use_kakao_talk_business; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_info.use_kakao_talk_business IS '카카오 알림톡 전송 여부';


--
-- Name: boost_receipt_info_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_receipt_info_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_receipt_info_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_receipt_info_id_seq OWNED BY public.boost_receipt_info.id;


--
-- Name: boost_receipt_payments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_receipt_payments (
    id bigint NOT NULL,
    receipt_id bigint NOT NULL,
    card_payment numeric(13,4) DEFAULT 0,
    payment_date timestamp without time zone,
    send_notification boolean DEFAULT false NOT NULL,
    notification_send_date timestamp without time zone,
    image_url character varying(512),
    message_id character varying,
    request_crawling boolean DEFAULT false NOT NULL,
    request_crawling_date timestamp without time zone,
    complete_crawling_date timestamp without time zone
);


--
-- Name: TABLE boost_receipt_payments; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.boost_receipt_payments IS '결제 내역 정보';


--
-- Name: COLUMN boost_receipt_payments.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_payments.id IS '결제 고유 식별자';


--
-- Name: COLUMN boost_receipt_payments.receipt_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_payments.receipt_id IS 'receipts 테이블의 외래 키';


--
-- Name: COLUMN boost_receipt_payments.card_payment; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_payments.card_payment IS '카드 수납액';


--
-- Name: COLUMN boost_receipt_payments.payment_date; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_payments.payment_date IS '수납 일자 (YYYY-MM-DD)';


--
-- Name: COLUMN boost_receipt_payments.send_notification; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_payments.send_notification IS '영수증 알림톡 발송 여부';


--
-- Name: COLUMN boost_receipt_payments.notification_send_date; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_payments.notification_send_date IS '영수증 알림톡 발송 일시';


--
-- Name: COLUMN boost_receipt_payments.image_url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipt_payments.image_url IS '영수증 image url';


--
-- Name: boost_receipt_payments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_receipt_payments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_receipt_payments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_receipt_payments_id_seq OWNED BY public.boost_receipt_payments.id;


--
-- Name: boost_receipts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_receipts (
    id bigint NOT NULL,
    hospital_id bigint,
    status integer,
    checkin_time timestamp without time zone,
    reservation_time time without time zone,
    doctor_id bigint,
    notes text,
    staff bigint,
    chair bigint,
    patient_id bigint NOT NULL,
    chart_number character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    is_new_patient boolean DEFAULT false,
    phone_number character varying(20),
    mobile_number character varying(20),
    gender boolean,
    birth_date date
);


--
-- Name: TABLE boost_receipts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.boost_receipts IS '진료 접수 및 환자 정보';


--
-- Name: COLUMN boost_receipts.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.id IS '진료 접수 고유 식별자';


--
-- Name: COLUMN boost_receipts.hospital_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.hospital_id IS '진료가 발생한 병원 식별자';


--
-- Name: COLUMN boost_receipts.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.status IS '접수 상태';


--
-- Name: COLUMN boost_receipts.checkin_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.checkin_time IS '접수 시각 (YYYY-MM-DD HH:MM:SS)';


--
-- Name: COLUMN boost_receipts.reservation_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.reservation_time IS '예약 시각 (HH:MM:SS)';


--
-- Name: COLUMN boost_receipts.doctor_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.doctor_id IS '담당의사 식별자';


--
-- Name: COLUMN boost_receipts.notes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.notes IS '접수 내용';


--
-- Name: COLUMN boost_receipts.staff; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.staff IS '담당직원 식별자';


--
-- Name: COLUMN boost_receipts.chair; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.chair IS '체어 번호';


--
-- Name: COLUMN boost_receipts.patient_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.patient_id IS '환자 고유 식별자';


--
-- Name: COLUMN boost_receipts.chart_number; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.chart_number IS '차트 번호';


--
-- Name: COLUMN boost_receipts.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.name IS '환자 이름';


--
-- Name: COLUMN boost_receipts.is_new_patient; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.is_new_patient IS '신환 여부 (0:false, 1:true)';


--
-- Name: COLUMN boost_receipts.phone_number; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.phone_number IS '전화번호';


--
-- Name: COLUMN boost_receipts.mobile_number; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.mobile_number IS '휴대폰 번호';


--
-- Name: COLUMN boost_receipts.gender; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.gender IS '성별 (0:false, 1:true)';


--
-- Name: COLUMN boost_receipts.birth_date; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.boost_receipts.birth_date IS '생년월일 (YYYY-MM-DD)';


--
-- Name: boost_receipts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_receipts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_receipts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_receipts_id_seq OWNED BY public.boost_receipts.id;


--
-- Name: boost_store_channel_list; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_store_channel_list (
    store_id bigint NOT NULL,
    channel_id character varying(255) NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);


--
-- Name: boost_store_info; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_store_info (
    id bigint NOT NULL,
    store_id bigint NOT NULL,
    store_name character varying(50) NOT NULL,
    title character varying(100),
    descriptions character varying(255),
    additional_comment character varying(100) DEFAULT ''::character varying,
    store_code character varying(20),
    use_custom_inquiry boolean DEFAULT false NOT NULL,
    present_url character varying(255)
);


--
-- Name: boost_store_info_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_store_info_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_store_info_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_store_info_id_seq OWNED BY public.boost_store_info.id;


--
-- Name: boost_team_gift; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_team_gift (
    id character varying NOT NULL,
    team_id character varying NOT NULL,
    gift_id character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: boost_team_message_template; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_team_message_template (
    id character varying NOT NULL,
    team_id character varying NOT NULL,
    message_template_id character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: boost_treatment_code; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_treatment_code (
    code character varying(6) NOT NULL,
    name character varying(10) NOT NULL,
    type character varying(6) NOT NULL
);


--
-- Name: boost_treatment_code_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_treatment_code_seq
    START WITH 1
    INCREMENT BY 1
    MINVALUE 0
    MAXVALUE 2176782335
    CACHE 1;


--
-- Name: boost_treatment_map; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_treatment_map (
    id bigint NOT NULL,
    code character varying(6) NOT NULL,
    store_id bigint NOT NULL
);


--
-- Name: boost_treatment_map_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_treatment_map_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: boost_treatment_map_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.boost_treatment_map_id_seq OWNED BY public.boost_treatment_map.id;


--
-- Name: boost_treatment_type_code; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.boost_treatment_type_code (
    code character varying(6) NOT NULL,
    name character varying(10) NOT NULL
);


--
-- Name: boost_treatment_type_code_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.boost_treatment_type_code_seq
    START WITH 1
    INCREMENT BY 1
    MINVALUE 0
    MAXVALUE 2176782335
    CACHE 1;


--
-- Name: channel; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.channel (
    id character varying(255) NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: COLUMN channel.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel.id IS 'NAVER_MAP, KAKAO_MAP, MODOODOC';


--
-- Name: channel_analysis_info; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.channel_analysis_info (
    id bigint NOT NULL,
    channel character varying(30) NOT NULL,
    analysis_type character varying(35) NOT NULL,
    user_prompt_template_id bigint NOT NULL,
    system_prompt_template_id bigint NOT NULL,
    ai_model_version_id bigint NOT NULL,
    usable boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: TABLE channel_analysis_info; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.channel_analysis_info IS '채널별 ai 분석 정보 관리';


--
-- Name: COLUMN channel_analysis_info.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.id IS 'id';


--
-- Name: COLUMN channel_analysis_info.channel; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.channel IS '채널';


--
-- Name: COLUMN channel_analysis_info.analysis_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.analysis_type IS '분석 종류(RELEVANCE:관련도/RELEVANCE_MATCH_REVIEW:관련도 병원일치 재검토)';


--
-- Name: COLUMN channel_analysis_info.user_prompt_template_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.user_prompt_template_id IS 'user prompt template id';


--
-- Name: COLUMN channel_analysis_info.system_prompt_template_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.system_prompt_template_id IS 'system prompt template id';


--
-- Name: COLUMN channel_analysis_info.ai_model_version_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.ai_model_version_id IS 'ai_model_version id';


--
-- Name: COLUMN channel_analysis_info.usable; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.usable IS '사용 여부';


--
-- Name: COLUMN channel_analysis_info.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.channel_analysis_info.created_at IS '생성일자';


--
-- Name: channel_analysis_info_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.channel_analysis_info_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: channel_analysis_info_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.channel_analysis_info_id_seq OWNED BY public.channel_analysis_info.id;


--
-- Name: community; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.community (
    id character varying NOT NULL,
    channel_id character varying NOT NULL,
    community_name character varying NOT NULL,
    community_url character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: COLUMN community.channel_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.community.channel_id IS 'NAVER_CAFE';


--
-- Name: community_keyword; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.community_keyword (
    id character varying NOT NULL,
    community_id character varying NOT NULL,
    keyword_id character varying NOT NULL,
    crawling_status character varying NOT NULL,
    crawling_start_at timestamp with time zone,
    crawling_end_at timestamp with time zone,
    success_crawling_from_time timestamp with time zone,
    crawling_fail_description character varying,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: COLUMN community_keyword.crawling_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.community_keyword.crawling_status IS '크롤링ING, STOP';


--
-- Name: consultation_request; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.consultation_request (
    id bigint NOT NULL,
    applicant_name character varying(50) NOT NULL,
    hospital_name character varying(100) NOT NULL,
    region character varying(100) NOT NULL,
    contact_number character varying(20) NOT NULL,
    email character varying(255) NOT NULL,
    interest_services integer DEFAULT 0 NOT NULL,
    description text NOT NULL,
    is_agreed_to_terms boolean NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: TABLE consultation_request; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.consultation_request IS '상담 신청 내역';


--
-- Name: COLUMN consultation_request.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.id IS '상담 신청 시퀀스 ID';


--
-- Name: COLUMN consultation_request.applicant_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.applicant_name IS '신청자 이름';


--
-- Name: COLUMN consultation_request.hospital_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.hospital_name IS '병원 명';


--
-- Name: COLUMN consultation_request.region; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.region IS '지역 (텍스트)';


--
-- Name: COLUMN consultation_request.contact_number; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.contact_number IS '연락처 (숫자만 저장, 하이픈 제외)';


--
-- Name: COLUMN consultation_request.email; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.email IS '이메일';


--
-- Name: COLUMN consultation_request.interest_services; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.interest_services IS '관심 서비스 다중 선택 (Bitmask 값)';


--
-- Name: COLUMN consultation_request.description; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.description IS '디테일한 상담 신청 내역';


--
-- Name: COLUMN consultation_request.is_agreed_to_terms; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.is_agreed_to_terms IS '약관 동의 여부';


--
-- Name: COLUMN consultation_request.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.consultation_request.created_at IS '신청 일자 (접수 시간)';


--
-- Name: consultation_request_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.consultation_request_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: consultation_request_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.consultation_request_id_seq OWNED BY public.consultation_request.id;


--
-- Name: consulting_request; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.consulting_request (
    id bigint NOT NULL,
    service character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    phone character varying(255) NOT NULL,
    email character varying(255),
    content character varying(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: consulting_request_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.consulting_request_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: consulting_request_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.consulting_request_id_seq OWNED BY public.consulting_request.id;


--
-- Name: crawler_account; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.crawler_account (
    id bigint NOT NULL,
    login_id character varying(255) NOT NULL,
    password character varying(255) NOT NULL,
    account_description character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone,
    usable boolean NOT NULL
);


--
-- Name: crawler_account_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.crawler_account_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: crawler_account_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.crawler_account_id_seq OWNED BY public.crawler_account.id;


--
-- Name: crawling_keyword; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.crawling_keyword (
    id bigint NOT NULL,
    name character varying NOT NULL,
    channel_id character varying NOT NULL,
    store_id bigint NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: crawling_keyword_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.crawling_keyword_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: crawling_keyword_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.crawling_keyword_id_seq OWNED BY public.crawling_keyword.id;


--
-- Name: crawling_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.crawling_log (
    id bigint NOT NULL,
    type character varying(20) NOT NULL,
    store_channel_id bigint,
    started_at timestamp with time zone NOT NULL,
    ended_at timestamp with time zone,
    crawling_result character varying(20),
    error_message character varying(255)
);


--
-- Name: crawling_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.crawling_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: crawling_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.crawling_log_id_seq OWNED BY public.crawling_log.id;


--
-- Name: flyway_schema_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flyway_schema_history (
    installed_rank integer NOT NULL,
    version character varying(50),
    description character varying(200) NOT NULL,
    type character varying(20) NOT NULL,
    script character varying(1000) NOT NULL,
    checksum integer,
    installed_by character varying(100) NOT NULL,
    installed_on timestamp without time zone DEFAULT now() NOT NULL,
    execution_time integer NOT NULL,
    success boolean NOT NULL
);


--
-- Name: hospital_treatment_positive_review; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.hospital_treatment_positive_review (
    hospital_id bigint NOT NULL,
    post_id character varying(255) NOT NULL,
    department character varying(100) NOT NULL,
    treatment character varying(255) NOT NULL,
    treatment_confidence double precision NOT NULL,
    sentiment character varying(30) NOT NULL,
    sentiment_confidence double precision NOT NULL,
    review_score double precision,
    content text,
    source character varying(255),
    analyzed_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: hospital_treatment_positive_review_summary; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.hospital_treatment_positive_review_summary (
    hospital_id bigint NOT NULL,
    department character varying(100) NOT NULL,
    treatment character varying(255) NOT NULL,
    review_count integer NOT NULL,
    positive_review_count integer NOT NULL,
    ratio double precision NOT NULL,
    review_ids text[] NOT NULL,
    examples jsonb DEFAULT '[]'::jsonb NOT NULL,
    source character varying(255),
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: kakao_alimtalk_button; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kakao_alimtalk_button (
    id bigint NOT NULL,
    template_id bigint NOT NULL,
    button_type character varying(2),
    button_name character varying(20),
    link_mobile character varying(1000),
    link_pc character varying(1000)
);


--
-- Name: kakao_alimtalk_button_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.kakao_alimtalk_button_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: kakao_alimtalk_button_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.kakao_alimtalk_button_id_seq OWNED BY public.kakao_alimtalk_button.id;


--
-- Name: kakao_alimtalk_template; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kakao_alimtalk_template (
    id bigint NOT NULL,
    code character varying(30) NOT NULL,
    template_code character varying(30) NOT NULL,
    contents character varying(1000) NOT NULL,
    plus_friend_id character varying(30) NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: kakao_alimtalk_template_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.kakao_alimtalk_template_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: kakao_alimtalk_template_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.kakao_alimtalk_template_id_seq OWNED BY public.kakao_alimtalk_template.id;


--
-- Name: keyword; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.keyword (
    id character varying NOT NULL,
    keyword_name character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: manual_crawl_source; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.manual_crawl_source (
    id bigint NOT NULL,
    url character varying NOT NULL,
    store_id bigint NOT NULL,
    community_id character varying(255) NOT NULL,
    keyword_id character varying(255) NOT NULL,
    community_keyword_id character varying(255) NOT NULL,
    account bigint NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone,
    usable boolean NOT NULL
);


--
-- Name: manual_crawl_source_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.manual_crawl_source_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: manual_crawl_source_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.manual_crawl_source_id_seq OWNED BY public.manual_crawl_source.id;


--
-- Name: match_review_analysis_data; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.match_review_analysis_data (
    id bigint NOT NULL,
    ai_analysis_log_id bigint NOT NULL,
    post_id character varying(25),
    thread_id character varying(40),
    published_at timestamp with time zone NOT NULL,
    store_name character varying(30) NOT NULL,
    review_type character varying(15),
    review_reason character varying(1000),
    result character varying(10),
    error_message character varying(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);


--
-- Name: TABLE match_review_analysis_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.match_review_analysis_data IS '관련도 병원일치 검토 데이터 관리';


--
-- Name: COLUMN match_review_analysis_data.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.id IS 'id';


--
-- Name: COLUMN match_review_analysis_data.ai_analysis_log_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.ai_analysis_log_id IS '분석 수행 로그 id';


--
-- Name: COLUMN match_review_analysis_data.post_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.post_id IS 'post id';


--
-- Name: COLUMN match_review_analysis_data.thread_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.thread_id IS 'thread id';


--
-- Name: COLUMN match_review_analysis_data.published_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.published_at IS '글 게시일자';


--
-- Name: COLUMN match_review_analysis_data.store_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.store_name IS '병원명';


--
-- Name: COLUMN match_review_analysis_data.review_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.review_type IS '검토 결과 유형 (CORRECT:정상,INCORRECT:오류)';


--
-- Name: COLUMN match_review_analysis_data.review_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.review_reason IS '분석 근거';


--
-- Name: COLUMN match_review_analysis_data.result; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.result IS '분석 결과 (SUCCESS:성공/FAIL:실패)';


--
-- Name: COLUMN match_review_analysis_data.error_message; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.error_message IS '오류 원인';


--
-- Name: COLUMN match_review_analysis_data.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.created_at IS '생성일자';


--
-- Name: COLUMN match_review_analysis_data.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.match_review_analysis_data.updated_at IS '수정일자';


--
-- Name: match_review_analysis_data_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.match_review_analysis_data_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: match_review_analysis_data_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.match_review_analysis_data_id_seq OWNED BY public.match_review_analysis_data.id;


--
-- Name: organization; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization (
    id character varying NOT NULL,
    organization_name character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: organization_app_user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_app_user (
    id character varying NOT NULL,
    organization_id character varying NOT NULL,
    app_user_id bigint NOT NULL,
    organization_role character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    organization_app_user_status character varying NOT NULL
);


--
-- Name: COLUMN organization_app_user.organization_role; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.organization_app_user.organization_role IS 'ORGANIZATION_OWNER,  ORGANIZATION_ADMIN, ORGANIZATION_USER, ORGANIZATION_TEMPORARY_USER';


--
-- Name: organization_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.organization_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: otp_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.otp_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: otp; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.otp (
    id bigint DEFAULT nextval('public.otp_id_seq'::regclass) NOT NULL,
    email character varying(255),
    phone character varying(255),
    otp_code character varying(255) NOT NULL,
    type character varying(255) NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: COLUMN otp.type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.otp.type IS 'LOGIN, JOIN';


--
-- Name: poll_checkpoint; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.poll_checkpoint (
    id character varying(50) NOT NULL,
    last_checked_at timestamp with time zone NOT NULL
);


--
-- Name: TABLE poll_checkpoint; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.poll_checkpoint IS 'cafe thread analysis 목적으로 Min 생성함';


--
-- Name: post; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post (
    id character varying(255) NOT NULL,
    score bigint,
    title character varying,
    content character varying,
    channel_id character varying(255),
    store_name character varying(255),
    store_detail character varying(500),
    author character varying(255),
    author_link character varying,
    author_at timestamp with time zone,
    capture_img character varying(255),
    usable boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    is_insulting boolean DEFAULT false NOT NULL,
    is_defamatory boolean DEFAULT false NOT NULL,
    post_reply_ai_recommend_id bigint,
    post_reply_id bigint,
    store_channel_id bigint,
    active_tags text[],
    treatment character varying,
    enhanced_content character varying,
    keywords text[],
    message_id character varying,
    can_reply boolean DEFAULT true NOT NULL
);


--
-- Name: post_classification; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_classification (
    id integer NOT NULL,
    post_id character varying,
    major_category character varying,
    minor_category character varying,
    content_match text,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: post_classification_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_classification_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_classification_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_classification_id_seq OWNED BY public.post_classification.id;


--
-- Name: post_classification_translation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_classification_translation (
    id bigint NOT NULL,
    post_classification_id bigint NOT NULL,
    locale character varying(10) NOT NULL,
    major_category character varying(255),
    minor_category character varying(255),
    content_match text,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    usable boolean DEFAULT true NOT NULL
);


--
-- Name: post_classification_translation_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_classification_translation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_classification_translation_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_classification_translation_id_seq OWNED BY public.post_classification_translation.id;


--
-- Name: post_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_reply (
    id bigint NOT NULL,
    post_id character varying(255),
    reply character varying,
    author character varying,
    author_dtm timestamp with time zone,
    usable boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    platform_id character varying
);


--
-- Name: post_reply_ai_recommend; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_reply_ai_recommend (
    id bigint NOT NULL,
    ai_recommendation_reply character varying,
    usable boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    post_id character varying
);


--
-- Name: post_reply_ai_recommend_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_reply_ai_recommend_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply_ai_recommend_id_seq1; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_reply_ai_recommend_id_seq1
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply_ai_recommend_id_seq1; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_reply_ai_recommend_id_seq1 OWNED BY public.post_reply_ai_recommend.id;


--
-- Name: post_reply_child_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_reply_child_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply_child; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_reply_child (
    id bigint DEFAULT nextval('public.post_reply_child_seq'::regclass) NOT NULL,
    platform_id character varying,
    content character varying,
    author character varying,
    author_at timestamp with time zone,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    post_reply_id bigint
);


--
-- Name: post_reply_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_reply_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_reply_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply_id_seq1; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_reply_id_seq1
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply_id_seq1; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_reply_id_seq1 OWNED BY public.post_reply.id;


--
-- Name: post_reply_registration_request_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_reply_registration_request_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_reply_registration_request; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_reply_registration_request (
    id bigint DEFAULT nextval('public.post_reply_registration_request_id_seq'::regclass) NOT NULL,
    post_id character varying(255) NOT NULL,
    team_id character varying,
    app_user_id bigint NOT NULL,
    reply character varying,
    status character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    request_result character varying
);


--
-- Name: COLUMN post_reply_registration_request.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.post_reply_registration_request.status IS 'PROGRESS, SUCCESS, FAIL';


--
-- Name: post_score; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_score (
    id bigint NOT NULL,
    post_id character varying(100) NOT NULL,
    content text NOT NULL,
    score double precision NOT NULL,
    sentiment character varying(20) NOT NULL,
    ad_score double precision,
    channel_id character varying(50),
    store_name character varying(255),
    author_dtm bigint,
    type character varying(50),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    adjusted_score double precision
);


--
-- Name: post_score_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_score_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_score_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_score_id_seq OWNED BY public.post_score.id;


--
-- Name: store_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.store_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: store; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store (
    id bigint DEFAULT nextval('public.store_id_seq'::regclass) NOT NULL,
    store_name character varying(255) NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    store_detail character varying(255),
    status character varying,
    store_image character varying,
    public_seo_enabled boolean DEFAULT false NOT NULL
);


--
-- Name: store_channel_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.store_channel_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: store_channel; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store_channel (
    id bigint DEFAULT nextval('public.store_channel_id_seq'::regclass) NOT NULL,
    channel_id character varying(255) NOT NULL,
    store_name character varying(255),
    store_detail character varying(255),
    crawling_start_at timestamp with time zone,
    crawling_end_at timestamp with time zone,
    status character varying(255) NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    store_id bigint,
    match_score numeric,
    success_crawling_from_time timestamp with time zone,
    channel_place_id character varying,
    description character varying
);


--
-- Name: COLUMN store_channel.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_channel.status IS 'SUCCESS, FAIL, PROGRESS, NOTIFICATION';


--
-- Name: COLUMN store_channel.success_crawling_from_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_channel.success_crawling_from_time IS '성공한 크롤링의 시작 시간';


--
-- Name: store_thread; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store_thread (
    id character varying NOT NULL,
    store_id bigint NOT NULL,
    thread_id character varying NOT NULL,
    display_status character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: thread; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.thread (
    id character varying NOT NULL,
    community_id character varying NOT NULL,
    title character varying,
    content character varying NOT NULL,
    author character varying NOT NULL,
    thread_url character varying NOT NULL,
    author_at timestamp with time zone NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    capture_img character varying
);


--
-- Name: post_thread_union_view; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.post_thread_union_view AS
 SELECT p.id,
    true AS is_post,
    p.score AS post_score,
    p.title AS post_title,
    p.content,
    p.channel_id,
    p.store_name,
    p.store_detail,
    p.author,
    p.author_link AS link,
    p.author_at,
    p.capture_img AS post_capture_img,
    p.usable,
    p.created_at AS post_created_at,
    p.updated_at AS post_updated_at,
    p.is_insulting AS post_is_insulting,
    p.is_defamatory AS post_is_defamatory,
    p.post_reply_ai_recommend_id,
    p.post_reply_id,
    p.store_channel_id,
    p.active_tags AS post_active_tags,
    p.treatment AS post_treatment,
    p.enhanced_content AS post_enhanced_content,
    p.keywords AS post_keywords,
    p.message_id AS post_message_id,
    p.can_reply AS post_can_reply,
    pr.reply,
    pr.author_dtm AS reply_at,
    prai.ai_recommendation_reply AS airecommendationreply,
    sc.crawling_start_at,
    sc.crawling_end_at,
    sc.usable AS store_channel_usable,
    sc.created_at AS store_channel_created_at,
    sc.updated_at AS store_channel_updated_at,
    sc.store_id,
    sc.match_score,
    sc.success_crawling_from_time,
    sc.channel_place_id,
    sc.description AS store_channel_description,
    ''::character varying AS store_thread_id,
    ''::character varying AS display_status,
    NULL::boolean AS store_thread_usable,
    NULL::timestamp with time zone AS store_thread_created_at,
    NULL::timestamp with time zone AS store_thread_updated_at,
    ''::character varying AS community_id,
    ''::character varying AS thread_title,
    NULL::timestamp with time zone AS thread_created_at,
    NULL::timestamp with time zone AS thread_updated_at,
    ''::character varying AS capture_img,
    ''::character varying AS community_name,
    ''::character varying AS community_url,
    NULL::boolean AS community_usable,
    NULL::timestamp with time zone AS community_created_at,
    NULL::timestamp with time zone AS community_updated_at
   FROM (((public.post p
     LEFT JOIN public.store_channel sc ON ((sc.id = p.store_channel_id)))
     LEFT JOIN public.post_reply pr ON (((p.id)::text = (pr.post_id)::text)))
     LEFT JOIN public.post_reply_ai_recommend prai ON ((p.post_reply_ai_recommend_id = prai.id)))
  WHERE (p.usable = true)
UNION ALL
 SELECT st.thread_id AS id,
    false AS is_post,
    NULL::bigint AS post_score,
    ''::character varying AS post_title,
    t.content,
    c.channel_id,
    s.store_name,
    s.store_detail,
    t.author,
    t.thread_url AS link,
    t.author_at,
    ''::character varying AS post_capture_img,
    t.usable,
    NULL::timestamp with time zone AS post_created_at,
    NULL::timestamp with time zone AS post_updated_at,
    NULL::boolean AS post_is_insulting,
    NULL::boolean AS post_is_defamatory,
    NULL::integer AS post_reply_ai_recommend_id,
    NULL::integer AS post_reply_id,
    NULL::integer AS store_channel_id,
    '{}'::text[] AS post_active_tags,
    ''::character varying AS post_treatment,
    ''::character varying AS post_enhanced_content,
    '{}'::text[] AS post_keywords,
    ''::character varying AS post_message_id,
    NULL::boolean AS post_can_reply,
    ''::character varying AS reply,
    NULL::timestamp with time zone AS reply_at,
    ''::character varying AS airecommendationreply,
    NULL::timestamp with time zone AS crawling_start_at,
    NULL::timestamp with time zone AS crawling_end_at,
    NULL::boolean AS store_channel_usable,
    NULL::timestamp with time zone AS store_channel_created_at,
    NULL::timestamp with time zone AS store_channel_updated_at,
    st.store_id,
    NULL::numeric AS match_score,
    NULL::timestamp with time zone AS success_crawling_from_time,
    NULL::character varying AS channel_place_id,
    ''::character varying AS store_channel_description,
    st.id AS store_thread_id,
    st.display_status,
    st.usable AS store_thread_usable,
    st.created_at AS store_thread_created_at,
    st.updated_at AS store_thread_updated_at,
    t.community_id,
    t.title AS thread_title,
    t.created_at AS thread_created_at,
    t.updated_at AS thread_updated_at,
    t.capture_img,
    c.community_name,
    c.community_url,
    c.usable AS community_usable,
    c.created_at AS community_created_at,
    c.updated_at AS community_updated_at
   FROM (((public.store_thread st
     LEFT JOIN public.thread t ON (((st.thread_id)::text = (t.id)::text)))
     LEFT JOIN public.community c ON (((t.community_id)::text = (c.id)::text)))
     LEFT JOIN public.store s ON ((s.id = st.store_id)))
  WHERE ((st.display_status)::text = 'VISIBLE'::text);


--
-- Name: COLUMN post_thread_union_view.success_crawling_from_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.post_thread_union_view.success_crawling_from_time IS '성공한 크롤링의 시작 시간';


--
-- Name: post_treatment_translation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_treatment_translation (
    id bigint NOT NULL,
    post_id character varying(255) NOT NULL,
    treatment character varying(255) NOT NULL,
    locale character varying(10) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    usable boolean DEFAULT true NOT NULL
);


--
-- Name: post_treatment_translation_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_treatment_translation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_treatment_translation_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_treatment_translation_id_seq OWNED BY public.post_treatment_translation.id;


--
-- Name: post_word; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_word (
    id bigint NOT NULL,
    post_id character varying(255),
    content character varying(500),
    word character varying(255),
    type character varying(255),
    usable boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    content_first_index integer,
    content_last_index integer
);


--
-- Name: post_word_attribute; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.post_word_attribute (
    id bigint NOT NULL,
    post_word_id bigint NOT NULL,
    attribute character varying(50) NOT NULL,
    type character varying(10) NOT NULL,
    attribute_column character varying(50) NOT NULL,
    attribute_value text NOT NULL
);


--
-- Name: post_word_attribute_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_word_attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_word_attribute_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_word_attribute_id_seq OWNED BY public.post_word_attribute.id;


--
-- Name: post_word_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_word_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_word_id_seq1; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.post_word_id_seq1
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: post_word_id_seq1; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.post_word_id_seq1 OWNED BY public.post_word.id;


--
-- Name: product_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.product_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: product; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product (
    id bigint DEFAULT nextval('public.product_id_seq'::regclass) NOT NULL,
    product_name character varying(255) NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    price integer NOT NULL
);


--
-- Name: product_post_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.product_post_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: product_post; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_post (
    id bigint DEFAULT nextval('public.product_post_id_seq'::regclass) NOT NULL,
    app_user_id bigint NOT NULL,
    post_id character varying(255),
    product_id bigint NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    status character varying(255) NOT NULL,
    original_title character varying,
    original_content character varying,
    original_channel_id character varying,
    original_store_name character varying,
    original_store_detail character varying,
    original_author character varying,
    original_author_link character varying,
    original_author_at timestamp with time zone,
    original_capture_img character varying,
    team_id character varying,
    thread_id character varying(255)
);


--
-- Name: prompt_template; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prompt_template (
    id bigint NOT NULL,
    prompt_type character varying(10) NOT NULL,
    template text NOT NULL,
    usable boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: TABLE prompt_template; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.prompt_template IS 'ai 프롬프트 관리';


--
-- Name: COLUMN prompt_template.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.prompt_template.id IS 'id';


--
-- Name: COLUMN prompt_template.prompt_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.prompt_template.prompt_type IS '템플릿 종류(SYSTEM/USER)';


--
-- Name: COLUMN prompt_template.template; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.prompt_template.template IS '메세지 양식';


--
-- Name: COLUMN prompt_template.usable; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.prompt_template.usable IS '사용 여부';


--
-- Name: COLUMN prompt_template.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.prompt_template.created_at IS '생성일자';


--
-- Name: prompt_template_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.prompt_template_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: prompt_template_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.prompt_template_id_seq OWNED BY public.prompt_template.id;


--
-- Name: relevance_analysis_data; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.relevance_analysis_data (
    id bigint NOT NULL,
    ai_analysis_log_id bigint NOT NULL,
    post_id character varying(25),
    thread_id character varying(40),
    published_at timestamp with time zone NOT NULL,
    store_name character varying(30) NOT NULL,
    relevance_type character varying(15),
    relevance_score character varying(25),
    relevance_reason character varying(1000),
    is_display boolean DEFAULT false NOT NULL,
    result character varying(10),
    error_message character varying(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);


--
-- Name: TABLE relevance_analysis_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.relevance_analysis_data IS '관련도 분석 데이터 관리';


--
-- Name: COLUMN relevance_analysis_data.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.id IS 'id';


--
-- Name: COLUMN relevance_analysis_data.ai_analysis_log_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.ai_analysis_log_id IS '분석 수행 로그 id';


--
-- Name: COLUMN relevance_analysis_data.post_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.post_id IS 'post id';


--
-- Name: COLUMN relevance_analysis_data.thread_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.thread_id IS 'thread id';


--
-- Name: COLUMN relevance_analysis_data.published_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.published_at IS '글 게시일자';


--
-- Name: COLUMN relevance_analysis_data.store_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.store_name IS '병원명';


--
-- Name: COLUMN relevance_analysis_data.relevance_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.relevance_type IS '관련도 유형 (MATCH:병원일치/MONITORING:모니터링/UNRELATED:관련도없음)';


--
-- Name: COLUMN relevance_analysis_data.relevance_score; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.relevance_score IS '관련도 점수 (HIGH_CONFIDENCE:확실3,MEDIUM_CONFIDENCE:중간2,LOW_CONFIDENCE:불확실1)';


--
-- Name: COLUMN relevance_analysis_data.relevance_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.relevance_reason IS '분석 근거';


--
-- Name: COLUMN relevance_analysis_data.is_display; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.is_display IS '노출 여부';


--
-- Name: COLUMN relevance_analysis_data.result; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.result IS '분석 결과 (SUCCESS:성공/FAIL:실패)';


--
-- Name: COLUMN relevance_analysis_data.error_message; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.error_message IS '오류 원인';


--
-- Name: COLUMN relevance_analysis_data.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.created_at IS '생성일자';


--
-- Name: COLUMN relevance_analysis_data.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.relevance_analysis_data.updated_at IS '수정일자';


--
-- Name: relevance_analysis_data_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.relevance_analysis_data_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: relevance_analysis_data_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.relevance_analysis_data_id_seq OWNED BY public.relevance_analysis_data.id;


--
-- Name: seq_address_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_address_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_app_user_channel_account; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_app_user_channel_account
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_app_user_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_app_user_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_organization_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_organization_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_otp_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_otp_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_post_reply_ai_recommend_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_post_reply_ai_recommend_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_post_reply_history_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_post_reply_history_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_post_reply_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_post_reply_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_post_reply_registration_request_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_post_reply_registration_request_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_post_word_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_post_word_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_product_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_product_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_product_post_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_product_post_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_store; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_store
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_team_address_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_team_address_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_team_app_user_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_team_app_user_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_team_channel_account_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_team_channel_account_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_team_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_team_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seq_user_address_id; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.seq_user_address_id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: similarity_score_analysis_data; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.similarity_score_analysis_data (
    id bigint NOT NULL,
    ai_analysis_log_id bigint NOT NULL,
    target_post_id character varying(50) NOT NULL,
    compare_target_post_id character varying(50) NOT NULL,
    score integer DEFAULT 0 NOT NULL,
    is_similar boolean DEFAULT false NOT NULL,
    result character varying(10),
    error_message character varying(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);


--
-- Name: TABLE similarity_score_analysis_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.similarity_score_analysis_data IS '유사도 분석 데이터 관리';


--
-- Name: COLUMN similarity_score_analysis_data.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.id IS 'id';


--
-- Name: COLUMN similarity_score_analysis_data.ai_analysis_log_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.ai_analysis_log_id IS '분석 수행 로그 id';


--
-- Name: COLUMN similarity_score_analysis_data.target_post_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.target_post_id IS 'post id';


--
-- Name: COLUMN similarity_score_analysis_data.compare_target_post_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.compare_target_post_id IS '유사도 비교 post id';


--
-- Name: COLUMN similarity_score_analysis_data.score; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.score IS '유사도 점수';


--
-- Name: COLUMN similarity_score_analysis_data.is_similar; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.is_similar IS '유사 여부';


--
-- Name: COLUMN similarity_score_analysis_data.result; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.result IS '분석 결과';


--
-- Name: COLUMN similarity_score_analysis_data.error_message; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.error_message IS '오류 원인';


--
-- Name: COLUMN similarity_score_analysis_data.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.created_at IS '생성일자';


--
-- Name: COLUMN similarity_score_analysis_data.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.similarity_score_analysis_data.updated_at IS '수정일자';


--
-- Name: similarity_score_analysis_data_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.similarity_score_analysis_data_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: similarity_score_analysis_data_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.similarity_score_analysis_data_id_seq OWNED BY public.similarity_score_analysis_data.id;


--
-- Name: store_channel_attribute_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.store_channel_attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: store_channel_attribute; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store_channel_attribute (
    id bigint DEFAULT nextval('public.store_channel_attribute_id_seq'::regclass) NOT NULL,
    store_channel bigint NOT NULL,
    attribute character varying NOT NULL,
    value character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: store_community; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store_community (
    id character varying NOT NULL,
    store_id bigint NOT NULL,
    community_id character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: store_community_keyword; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store_community_keyword (
    id character varying NOT NULL,
    store_id bigint NOT NULL,
    community_id character varying NOT NULL,
    keyword_id character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: store_relevance_info; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store_relevance_info (
    id bigint NOT NULL,
    store_id bigint NOT NULL,
    store_name character varying(30) NOT NULL,
    notice character varying(2000),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);


--
-- Name: TABLE store_relevance_info; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.store_relevance_info IS '병원별 ai 분석 정보 관리';


--
-- Name: COLUMN store_relevance_info.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_info.id IS 'id';


--
-- Name: COLUMN store_relevance_info.store_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_info.store_id IS '병원 id';


--
-- Name: COLUMN store_relevance_info.store_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_info.store_name IS '병원명';


--
-- Name: COLUMN store_relevance_info.notice; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_info.notice IS '병원 관련 주의사항 정보';


--
-- Name: COLUMN store_relevance_info.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_info.created_at IS '생성일자';


--
-- Name: COLUMN store_relevance_info.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_info.updated_at IS '수정일자';


--
-- Name: store_relevance_info_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.store_relevance_info_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: store_relevance_info_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.store_relevance_info_id_seq OWNED BY public.store_relevance_info.id;


--
-- Name: store_relevance_keyword; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.store_relevance_keyword (
    id bigint NOT NULL,
    store_relevance_info_id bigint NOT NULL,
    code character varying(20) NOT NULL,
    value character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);


--
-- Name: TABLE store_relevance_keyword; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.store_relevance_keyword IS '병원별 ai 분석 키워드 관리';


--
-- Name: COLUMN store_relevance_keyword.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_keyword.id IS 'id';


--
-- Name: COLUMN store_relevance_keyword.store_relevance_info_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_keyword.store_relevance_info_id IS '병원별 분석 정보 id';


--
-- Name: COLUMN store_relevance_keyword.code; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_keyword.code IS '키워드 유형 (LOCATION:지역/SPECIALTY:진료/MONITORING:모니터링)';


--
-- Name: COLUMN store_relevance_keyword.value; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_keyword.value IS '키워드';


--
-- Name: COLUMN store_relevance_keyword.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_keyword.created_at IS '생성일자';


--
-- Name: COLUMN store_relevance_keyword.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.store_relevance_keyword.updated_at IS '수정일자';


--
-- Name: store_relevance_keyword_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.store_relevance_keyword_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: store_relevance_keyword_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.store_relevance_keyword_id_seq OWNED BY public.store_relevance_keyword.id;


--
-- Name: subscribe_product; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscribe_product (
    id bigint NOT NULL,
    store_id bigint NOT NULL,
    product character varying(30) NOT NULL,
    status character varying(30) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);


--
-- Name: subscribe_product_attribute; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscribe_product_attribute (
    id bigint NOT NULL,
    subscribe_product_id bigint NOT NULL,
    key character varying(25) NOT NULL,
    value character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: subscribe_product_attribute_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.subscribe_product_attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: subscribe_product_attribute_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.subscribe_product_attribute_id_seq OWNED BY public.subscribe_product_attribute.id;


--
-- Name: subscribe_product_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscribe_product_history (
    id bigint NOT NULL,
    subscribe_product_id bigint NOT NULL,
    action character varying(30) NOT NULL,
    requester_id bigint NOT NULL,
    previous_product_detail character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: subscribe_product_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.subscribe_product_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: subscribe_product_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.subscribe_product_history_id_seq OWNED BY public.subscribe_product_history.id;


--
-- Name: subscribe_product_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.subscribe_product_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: subscribe_product_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.subscribe_product_id_seq OWNED BY public.subscribe_product.id;


--
-- Name: target_store; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.target_store (
    store_id bigint NOT NULL,
    store_name character varying,
    store_detail character varying
);


--
-- Name: team; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team (
    organization_id character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    store_id bigint NOT NULL,
    id character varying NOT NULL,
    phone character varying,
    unit_id bigint NOT NULL
);


--
-- Name: COLUMN team.store_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.team.store_id IS '일대일 관계';


--
-- Name: team_app_user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_app_user (
    id character varying NOT NULL,
    app_user_id bigint NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    permission_writing boolean DEFAULT true NOT NULL,
    permission_reading boolean DEFAULT true NOT NULL,
    organization_app_user_id character varying NOT NULL,
    team_id character varying NOT NULL,
    organization_id character varying
);


--
-- Name: team_app_user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.team_app_user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: team_auth; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_auth (
    id character varying NOT NULL,
    store_id bigint NOT NULL,
    app_user_id bigint NOT NULL,
    account_id character varying NOT NULL,
    account_password character varying NOT NULL,
    result character varying NOT NULL,
    description character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    channel_id character varying
);


--
-- Name: team_auth_bypass; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_auth_bypass (
    id bigint NOT NULL,
    app_user_id bigint NOT NULL,
    store_id bigint NOT NULL
);


--
-- Name: team_auth_bypass_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.team_auth_bypass_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: team_auth_bypass_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.team_auth_bypass_id_seq OWNED BY public.team_auth_bypass.id;


--
-- Name: team_channel_account; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_channel_account (
    id character varying NOT NULL,
    channel_id character varying NOT NULL,
    account_id character varying NOT NULL,
    account_password character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    app_user_id bigint NOT NULL,
    auth_success_at timestamp with time zone,
    auth_fail_at timestamp with time zone,
    login_result character varying,
    team_id character varying NOT NULL
);


--
-- Name: team_channel_account_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.team_channel_account_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: team_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.team_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: team_store_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.team_store_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: team_unit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.team_unit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: team_unit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.team_unit_id_seq OWNED BY public.team.unit_id;


--
-- Name: thread_comment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.thread_comment (
    id character varying NOT NULL,
    thread_id character varying NOT NULL,
    content character varying NOT NULL,
    author character varying NOT NULL,
    author_at timestamp with time zone NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    capture_img character varying,
    report_url character varying
);


--
-- Name: thread_keyword; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.thread_keyword (
    id character varying NOT NULL,
    thread_id character varying NOT NULL,
    keyword_id character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: thread_reply; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.thread_reply (
    id character varying NOT NULL,
    thread_comment_id character varying NOT NULL,
    content character varying NOT NULL,
    author character varying NOT NULL,
    author_at timestamp with time zone NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    capture_img character varying,
    report_url character varying
);


--
-- Name: token; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.token (
    refresh_token_key character varying NOT NULL,
    refresh_token_value character varying NOT NULL,
    usable boolean NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: ai_analysis_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ai_analysis_log ALTER COLUMN id SET DEFAULT nextval('public.ai_analysis_log_id_seq'::regclass);


--
-- Name: ai_model_version id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ai_model_version ALTER COLUMN id SET DEFAULT nextval('public.ai_model_version_id_seq'::regclass);


--
-- Name: app_user_attribute id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_attribute ALTER COLUMN id SET DEFAULT nextval('public.app_user_attribute_id_seq'::regclass);


--
-- Name: app_user_social id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_social ALTER COLUMN id SET DEFAULT nextval('public.app_user_social_id_seq'::regclass);


--
-- Name: boost_answer_icon id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_answer_icon ALTER COLUMN id SET DEFAULT nextval('public.boost_answer_icon_id_seq'::regclass);


--
-- Name: boost_answer_map id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_answer_map ALTER COLUMN id SET DEFAULT nextval('public.boost_answer_map_id_seq'::regclass);


--
-- Name: boost_dental_image id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_dental_image ALTER COLUMN id SET DEFAULT nextval('public.boost_dental_image_id_seq'::regclass);


--
-- Name: boost_dental_patient_info id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_dental_patient_info ALTER COLUMN id SET DEFAULT nextval('public.boost_dental_patient_info_id_seq'::regclass);


--
-- Name: boost_post_similarity id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_post_similarity ALTER COLUMN id SET DEFAULT nextval('public.boost_post_similarity_id_seq'::regclass);


--
-- Name: boost_present id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present ALTER COLUMN id SET DEFAULT nextval('public.boost_present_id_seq'::regclass);


--
-- Name: boost_present_ledger id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_ledger ALTER COLUMN id SET DEFAULT nextval('public.boost_present_ledger_id_seq'::regclass);


--
-- Name: boost_present_order id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_order ALTER COLUMN id SET DEFAULT nextval('public.boost_present_order_id_seq'::regclass);


--
-- Name: boost_present_order_attribute id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_order_attribute ALTER COLUMN id SET DEFAULT nextval('public.boost_present_order_attribute_id_seq'::regclass);


--
-- Name: boost_present_order_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_order_history ALTER COLUMN id SET DEFAULT nextval('public.boost_present_order_history_id_seq'::regclass);


--
-- Name: boost_question_map id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_question_map ALTER COLUMN id SET DEFAULT nextval('public.boost_question_map_id_seq'::regclass);


--
-- Name: boost_receipt_info id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_receipt_info ALTER COLUMN id SET DEFAULT nextval('public.boost_receipt_info_id_seq'::regclass);


--
-- Name: boost_receipt_payments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_receipt_payments ALTER COLUMN id SET DEFAULT nextval('public.boost_receipt_payments_id_seq'::regclass);


--
-- Name: boost_receipts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_receipts ALTER COLUMN id SET DEFAULT nextval('public.boost_receipts_id_seq'::regclass);


--
-- Name: boost_store_info id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_store_info ALTER COLUMN id SET DEFAULT nextval('public.boost_store_info_id_seq'::regclass);


--
-- Name: boost_treatment_map id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_treatment_map ALTER COLUMN id SET DEFAULT nextval('public.boost_treatment_map_id_seq'::regclass);


--
-- Name: channel_analysis_info id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.channel_analysis_info ALTER COLUMN id SET DEFAULT nextval('public.channel_analysis_info_id_seq'::regclass);


--
-- Name: consultation_request id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.consultation_request ALTER COLUMN id SET DEFAULT nextval('public.consultation_request_id_seq'::regclass);


--
-- Name: consulting_request id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.consulting_request ALTER COLUMN id SET DEFAULT nextval('public.consulting_request_id_seq'::regclass);


--
-- Name: crawler_account id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crawler_account ALTER COLUMN id SET DEFAULT nextval('public.crawler_account_id_seq'::regclass);


--
-- Name: crawling_keyword id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crawling_keyword ALTER COLUMN id SET DEFAULT nextval('public.crawling_keyword_id_seq'::regclass);


--
-- Name: crawling_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crawling_log ALTER COLUMN id SET DEFAULT nextval('public.crawling_log_id_seq'::regclass);


--
-- Name: kakao_alimtalk_button id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kakao_alimtalk_button ALTER COLUMN id SET DEFAULT nextval('public.kakao_alimtalk_button_id_seq'::regclass);


--
-- Name: kakao_alimtalk_template id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kakao_alimtalk_template ALTER COLUMN id SET DEFAULT nextval('public.kakao_alimtalk_template_id_seq'::regclass);


--
-- Name: manual_crawl_source id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.manual_crawl_source ALTER COLUMN id SET DEFAULT nextval('public.manual_crawl_source_id_seq'::regclass);


--
-- Name: match_review_analysis_data id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.match_review_analysis_data ALTER COLUMN id SET DEFAULT nextval('public.match_review_analysis_data_id_seq'::regclass);


--
-- Name: post_classification id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_classification ALTER COLUMN id SET DEFAULT nextval('public.post_classification_id_seq'::regclass);


--
-- Name: post_classification_translation id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_classification_translation ALTER COLUMN id SET DEFAULT nextval('public.post_classification_translation_id_seq'::regclass);


--
-- Name: post_reply id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply ALTER COLUMN id SET DEFAULT nextval('public.post_reply_id_seq1'::regclass);


--
-- Name: post_reply_ai_recommend id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_ai_recommend ALTER COLUMN id SET DEFAULT nextval('public.post_reply_ai_recommend_id_seq1'::regclass);


--
-- Name: post_score id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_score ALTER COLUMN id SET DEFAULT nextval('public.post_score_id_seq'::regclass);


--
-- Name: post_treatment_translation id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_treatment_translation ALTER COLUMN id SET DEFAULT nextval('public.post_treatment_translation_id_seq'::regclass);


--
-- Name: post_word id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_word ALTER COLUMN id SET DEFAULT nextval('public.post_word_id_seq1'::regclass);


--
-- Name: post_word_attribute id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_word_attribute ALTER COLUMN id SET DEFAULT nextval('public.post_word_attribute_id_seq'::regclass);


--
-- Name: prompt_template id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompt_template ALTER COLUMN id SET DEFAULT nextval('public.prompt_template_id_seq'::regclass);


--
-- Name: relevance_analysis_data id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.relevance_analysis_data ALTER COLUMN id SET DEFAULT nextval('public.relevance_analysis_data_id_seq'::regclass);


--
-- Name: similarity_score_analysis_data id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.similarity_score_analysis_data ALTER COLUMN id SET DEFAULT nextval('public.similarity_score_analysis_data_id_seq'::regclass);


--
-- Name: store_relevance_info id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_relevance_info ALTER COLUMN id SET DEFAULT nextval('public.store_relevance_info_id_seq'::regclass);


--
-- Name: store_relevance_keyword id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_relevance_keyword ALTER COLUMN id SET DEFAULT nextval('public.store_relevance_keyword_id_seq'::regclass);


--
-- Name: subscribe_product id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscribe_product ALTER COLUMN id SET DEFAULT nextval('public.subscribe_product_id_seq'::regclass);


--
-- Name: subscribe_product_attribute id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscribe_product_attribute ALTER COLUMN id SET DEFAULT nextval('public.subscribe_product_attribute_id_seq'::regclass);


--
-- Name: subscribe_product_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscribe_product_history ALTER COLUMN id SET DEFAULT nextval('public.subscribe_product_history_id_seq'::regclass);


--
-- Name: team unit_id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team ALTER COLUMN unit_id SET DEFAULT nextval('public.team_unit_id_seq'::regclass);


--
-- Name: team_auth_bypass id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_auth_bypass ALTER COLUMN id SET DEFAULT nextval('public.team_auth_bypass_id_seq'::regclass);


--
-- Name: store_channel PK_ADDRESS; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_channel
    ADD CONSTRAINT "PK_ADDRESS" PRIMARY KEY (id);


--
-- Name: app_user PK_APP_USER; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user
    ADD CONSTRAINT "PK_APP_USER" PRIMARY KEY (id);


--
-- Name: boost_customer PK_BOOST_CUSTOMER; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_customer
    ADD CONSTRAINT "PK_BOOST_CUSTOMER" PRIMARY KEY (id);


--
-- Name: boost_customer_gift PK_BOOST_CUSTOMER_GIFT; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_customer_gift
    ADD CONSTRAINT "PK_BOOST_CUSTOMER_GIFT" PRIMARY KEY (id);


--
-- Name: boost_gift PK_BOOST_GIFT; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_gift
    ADD CONSTRAINT "PK_BOOST_GIFT" PRIMARY KEY (id);


--
-- Name: boost_message PK_BOOST_MESSAGE_SEND; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_message
    ADD CONSTRAINT "PK_BOOST_MESSAGE_SEND" PRIMARY KEY (id);


--
-- Name: boost_message_template PK_BOOST_MESSAGE_TEMPLATE; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_message_template
    ADD CONSTRAINT "PK_BOOST_MESSAGE_TEMPLATE" PRIMARY KEY (id);


--
-- Name: boost_team_gift PK_BOOST_TEAM_GIFT; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_gift
    ADD CONSTRAINT "PK_BOOST_TEAM_GIFT" PRIMARY KEY (id);


--
-- Name: boost_team_message_template PK_BOOST_TEAM_MESSAGE_TEMPLATE; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_message_template
    ADD CONSTRAINT "PK_BOOST_TEAM_MESSAGE_TEMPLATE" PRIMARY KEY (id);


--
-- Name: channel PK_CHANNEL; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.channel
    ADD CONSTRAINT "PK_CHANNEL" PRIMARY KEY (id);


--
-- Name: community PK_COMMUNITY; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.community
    ADD CONSTRAINT "PK_COMMUNITY" PRIMARY KEY (id);


--
-- Name: community_keyword PK_COMMUNITY_KEYWORD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.community_keyword
    ADD CONSTRAINT "PK_COMMUNITY_KEYWORD" PRIMARY KEY (id);


--
-- Name: consulting_request PK_CONSULTING_REQUEST; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.consulting_request
    ADD CONSTRAINT "PK_CONSULTING_REQUEST" PRIMARY KEY (id);


--
-- Name: keyword PK_KEYWORD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keyword
    ADD CONSTRAINT "PK_KEYWORD" PRIMARY KEY (id);


--
-- Name: organization PK_ORGANIZATION; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization
    ADD CONSTRAINT "PK_ORGANIZATION" PRIMARY KEY (id);


--
-- Name: organization_app_user PK_ORGANIZATION_APP_USER; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_app_user
    ADD CONSTRAINT "PK_ORGANIZATION_APP_USER" PRIMARY KEY (id);


--
-- Name: team_app_user PK_ORGANIZATION_APP_USER_TEAM; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_app_user
    ADD CONSTRAINT "PK_ORGANIZATION_APP_USER_TEAM" PRIMARY KEY (id);


--
-- Name: otp PK_OTP; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otp
    ADD CONSTRAINT "PK_OTP" PRIMARY KEY (id);


--
-- Name: post PK_POST; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post
    ADD CONSTRAINT "PK_POST" PRIMARY KEY (id);


--
-- Name: post_reply PK_POST_REPLY; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply
    ADD CONSTRAINT "PK_POST_REPLY" PRIMARY KEY (id);


--
-- Name: post_reply_ai_recommend PK_POST_REPLY_AI_RECOMMEND; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_ai_recommend
    ADD CONSTRAINT "PK_POST_REPLY_AI_RECOMMEND" PRIMARY KEY (id);


--
-- Name: post_reply_child PK_POST_REPLY_CHILD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_child
    ADD CONSTRAINT "PK_POST_REPLY_CHILD" PRIMARY KEY (id);


--
-- Name: post_reply_registration_request PK_POST_REPLY_REGISTRATION_REQUEST; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_registration_request
    ADD CONSTRAINT "PK_POST_REPLY_REGISTRATION_REQUEST" PRIMARY KEY (id);


--
-- Name: post_word PK_POST_WORD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_word
    ADD CONSTRAINT "PK_POST_WORD" PRIMARY KEY (id);


--
-- Name: product PK_PRODUCT; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product
    ADD CONSTRAINT "PK_PRODUCT" PRIMARY KEY (id);


--
-- Name: product_post PK_PRODUCT_POST; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_post
    ADD CONSTRAINT "PK_PRODUCT_POST" PRIMARY KEY (id);


--
-- Name: store PK_STORE; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store
    ADD CONSTRAINT "PK_STORE" PRIMARY KEY (id);


--
-- Name: store_channel_attribute PK_STORE_CHANNEL_ATTRIBUTE; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_channel_attribute
    ADD CONSTRAINT "PK_STORE_CHANNEL_ATTRIBUTE" PRIMARY KEY (id);


--
-- Name: store_community PK_STORE_COMMUNITY; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_community
    ADD CONSTRAINT "PK_STORE_COMMUNITY" PRIMARY KEY (id);


--
-- Name: store_community_keyword PK_STORE_COMMUNITY_KEYWORD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_community_keyword
    ADD CONSTRAINT "PK_STORE_COMMUNITY_KEYWORD" PRIMARY KEY (id);


--
-- Name: store_thread PK_STORE_THREAD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_thread
    ADD CONSTRAINT "PK_STORE_THREAD" PRIMARY KEY (id);


--
-- Name: team PK_TEAM; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team
    ADD CONSTRAINT "PK_TEAM" PRIMARY KEY (id);


--
-- Name: team_auth PK_TEAM_AUTH; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_auth
    ADD CONSTRAINT "PK_TEAM_AUTH" PRIMARY KEY (id);


--
-- Name: team_channel_account PK_TEAM_CHANNEL_ACCOUNT; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_channel_account
    ADD CONSTRAINT "PK_TEAM_CHANNEL_ACCOUNT" PRIMARY KEY (id);


--
-- Name: thread PK_THREAD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread
    ADD CONSTRAINT "PK_THREAD" PRIMARY KEY (id);


--
-- Name: thread_comment PK_THREAD_COMMENT; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread_comment
    ADD CONSTRAINT "PK_THREAD_COMMENT" PRIMARY KEY (id);


--
-- Name: thread_keyword PK_THREAD_KEYWORD; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread_keyword
    ADD CONSTRAINT "PK_THREAD_KEYWORD" PRIMARY KEY (id);


--
-- Name: thread_reply PK_THREAD_REPLY; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread_reply
    ADD CONSTRAINT "PK_THREAD_REPLY" PRIMARY KEY (id);


--
-- Name: token PK_TOKEN; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.token
    ADD CONSTRAINT "PK_TOKEN" PRIMARY KEY (refresh_token_key);


--
-- Name: app_user_store PK_USER_ADDRESS; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_store
    ADD CONSTRAINT "PK_USER_ADDRESS" PRIMARY KEY (id);


--
-- Name: ai_analysis_log ai_analysis_log_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ai_analysis_log
    ADD CONSTRAINT ai_analysis_log_pk PRIMARY KEY (id);


--
-- Name: ai_model_version ai_model_version_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ai_model_version
    ADD CONSTRAINT ai_model_version_pk PRIMARY KEY (id);


--
-- Name: app_user_attribute app_user_attribute_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_attribute
    ADD CONSTRAINT app_user_attribute_pkey PRIMARY KEY (id);


--
-- Name: app_user app_user_email; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user
    ADD CONSTRAINT app_user_email UNIQUE (email);


--
-- Name: boost_answer_code boost_answer_code_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_answer_code
    ADD CONSTRAINT boost_answer_code_pk PRIMARY KEY (code);


--
-- Name: boost_answer_map boost_answer_map_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_answer_map
    ADD CONSTRAINT boost_answer_map_pk PRIMARY KEY (id);


--
-- Name: boost_dental_image boost_dental_image_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_dental_image
    ADD CONSTRAINT boost_dental_image_pk PRIMARY KEY (id);


--
-- Name: boost_dental_patient_info boost_dental_patient_info_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_dental_patient_info
    ADD CONSTRAINT boost_dental_patient_info_pk PRIMARY KEY (id);


--
-- Name: boost_message_template boost_message_template_template_code_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_message_template
    ADD CONSTRAINT boost_message_template_template_code_unique UNIQUE (template_code);


--
-- Name: boost_present_ledger boost_present_ledger_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_ledger
    ADD CONSTRAINT boost_present_ledger_pk PRIMARY KEY (id);


--
-- Name: boost_present_order_attribute boost_present_order_attribute_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_order_attribute
    ADD CONSTRAINT boost_present_order_attribute_pk PRIMARY KEY (id);


--
-- Name: boost_present_order_history boost_present_order_history_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_order_history
    ADD CONSTRAINT boost_present_order_history_pk PRIMARY KEY (id);


--
-- Name: boost_present_order boost_present_order_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present_order
    ADD CONSTRAINT boost_present_order_pk PRIMARY KEY (id);


--
-- Name: boost_present boost_present_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_present
    ADD CONSTRAINT boost_present_pk PRIMARY KEY (id);


--
-- Name: boost_question_code boost_question_code_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_question_code
    ADD CONSTRAINT boost_question_code_pk PRIMARY KEY (code);


--
-- Name: boost_question_map boost_question_map_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_question_map
    ADD CONSTRAINT boost_question_map_pk PRIMARY KEY (id);


--
-- Name: boost_receipt_info boost_receipt_info_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_receipt_info
    ADD CONSTRAINT boost_receipt_info_pk PRIMARY KEY (id);


--
-- Name: boost_receipt_payments boost_receipt_payments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_receipt_payments
    ADD CONSTRAINT boost_receipt_payments_pkey PRIMARY KEY (id);


--
-- Name: boost_receipts boost_receipts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_receipts
    ADD CONSTRAINT boost_receipts_pkey PRIMARY KEY (id);


--
-- Name: boost_store_channel_list boost_store_channel_list_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_store_channel_list
    ADD CONSTRAINT boost_store_channel_list_pk PRIMARY KEY (store_id, channel_id);


--
-- Name: boost_store_info boost_store_info_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_store_info
    ADD CONSTRAINT boost_store_info_pk PRIMARY KEY (id);


--
-- Name: boost_team_gift boost_team_gift_team_id_gift_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_gift
    ADD CONSTRAINT boost_team_gift_team_id_gift_id_unique UNIQUE (team_id, gift_id);


--
-- Name: boost_team_message_template boost_team_message_template_team_id_message_template_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_message_template
    ADD CONSTRAINT boost_team_message_template_team_id_message_template_id_unique UNIQUE (team_id, message_template_id);


--
-- Name: boost_treatment_code boost_treatment_code_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_treatment_code
    ADD CONSTRAINT boost_treatment_code_pk PRIMARY KEY (code);


--
-- Name: boost_treatment_map boost_treatment_map_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_treatment_map
    ADD CONSTRAINT boost_treatment_map_pk PRIMARY KEY (id);


--
-- Name: boost_treatment_type_code boost_treatment_type_code_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_treatment_type_code
    ADD CONSTRAINT boost_treatment_type_code_pk PRIMARY KEY (code);


--
-- Name: channel_analysis_info channel_analysis_info_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.channel_analysis_info
    ADD CONSTRAINT channel_analysis_info_pk PRIMARY KEY (id);


--
-- Name: consultation_request consultation_request_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.consultation_request
    ADD CONSTRAINT consultation_request_pkey PRIMARY KEY (id);


--
-- Name: crawler_account crawler_account_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crawler_account
    ADD CONSTRAINT crawler_account_pkey PRIMARY KEY (id);


--
-- Name: crawling_keyword crawling_keyword_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crawling_keyword
    ADD CONSTRAINT crawling_keyword_pkey PRIMARY KEY (id);


--
-- Name: crawling_log crawling_log_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crawling_log
    ADD CONSTRAINT crawling_log_pk PRIMARY KEY (id);


--
-- Name: flyway_schema_history flyway_schema_history_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flyway_schema_history
    ADD CONSTRAINT flyway_schema_history_pk PRIMARY KEY (installed_rank);


--
-- Name: hospital_treatment_positive_review hospital_treatment_positive_review_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hospital_treatment_positive_review
    ADD CONSTRAINT hospital_treatment_positive_review_pkey PRIMARY KEY (hospital_id, post_id, department, treatment);


--
-- Name: hospital_treatment_positive_review_summary hospital_treatment_positive_review_summary_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hospital_treatment_positive_review_summary
    ADD CONSTRAINT hospital_treatment_positive_review_summary_pkey PRIMARY KEY (hospital_id, department, treatment);


--
-- Name: kakao_alimtalk_button kakao_alimtalk_button_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kakao_alimtalk_button
    ADD CONSTRAINT kakao_alimtalk_button_pkey PRIMARY KEY (id);


--
-- Name: kakao_alimtalk_template kakao_alimtalk_template_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kakao_alimtalk_template
    ADD CONSTRAINT kakao_alimtalk_template_pkey PRIMARY KEY (id);


--
-- Name: manual_crawl_source manual_crawl_source_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.manual_crawl_source
    ADD CONSTRAINT manual_crawl_source_pkey PRIMARY KEY (id);


--
-- Name: match_review_analysis_data match_review_analysis_data_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.match_review_analysis_data
    ADD CONSTRAINT match_review_analysis_data_pk PRIMARY KEY (id);


--
-- Name: team_app_user organization_app_user_team_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_app_user
    ADD CONSTRAINT organization_app_user_team_unique UNIQUE (organization_app_user_id, team_id);


--
-- Name: poll_checkpoint poll_checkpoint_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_checkpoint
    ADD CONSTRAINT poll_checkpoint_pkey PRIMARY KEY (id);


--
-- Name: post_classification post_classification_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_classification
    ADD CONSTRAINT post_classification_pkey PRIMARY KEY (id);


--
-- Name: post_classification_translation post_classification_translation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_classification_translation
    ADD CONSTRAINT post_classification_translation_pkey PRIMARY KEY (id);


--
-- Name: boost_post_similarity post_enhancement_similarity_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_post_similarity
    ADD CONSTRAINT post_enhancement_similarity_pk PRIMARY KEY (id);


--
-- Name: post_score post_score_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_score
    ADD CONSTRAINT post_score_pkey PRIMARY KEY (id);


--
-- Name: post_treatment_translation post_treatment_translation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_treatment_translation
    ADD CONSTRAINT post_treatment_translation_pkey PRIMARY KEY (id);


--
-- Name: post_word_attribute post_word_attribute_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_word_attribute
    ADD CONSTRAINT post_word_attribute_pkey PRIMARY KEY (id);


--
-- Name: prompt_template prompt_template_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompt_template
    ADD CONSTRAINT prompt_template_pk PRIMARY KEY (id);


--
-- Name: relevance_analysis_data relevance_analysis_data_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.relevance_analysis_data
    ADD CONSTRAINT relevance_analysis_data_pk PRIMARY KEY (id);


--
-- Name: similarity_score_analysis_data similarity_score_analysis_data_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.similarity_score_analysis_data
    ADD CONSTRAINT similarity_score_analysis_data_pk PRIMARY KEY (id);


--
-- Name: store_channel store_channel_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_channel
    ADD CONSTRAINT store_channel_unique UNIQUE (channel_id, store_name, store_detail);


--
-- Name: store_relevance_info store_relevance_info_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_relevance_info
    ADD CONSTRAINT store_relevance_info_pk PRIMARY KEY (id);


--
-- Name: store_relevance_keyword store_relevance_keyword_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_relevance_keyword
    ADD CONSTRAINT store_relevance_keyword_pk PRIMARY KEY (id);


--
-- Name: store stores_name_detail_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store
    ADD CONSTRAINT stores_name_detail_unique UNIQUE (store_name, store_detail);


--
-- Name: subscribe_product_attribute subscribe_product_attribute_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscribe_product_attribute
    ADD CONSTRAINT subscribe_product_attribute_pk PRIMARY KEY (id);


--
-- Name: subscribe_product_history subscribe_product_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscribe_product_history
    ADD CONSTRAINT subscribe_product_history_pkey PRIMARY KEY (id);


--
-- Name: subscribe_product subscribe_product_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscribe_product
    ADD CONSTRAINT subscribe_product_pkey PRIMARY KEY (id);


--
-- Name: target_store target_store_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.target_store
    ADD CONSTRAINT target_store_pkey PRIMARY KEY (store_id);


--
-- Name: product_post team_id_post_id_product_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_post
    ADD CONSTRAINT team_id_post_id_product_id_unique UNIQUE (team_id, post_id, product_id);


--
-- Name: store_thread uk_store_thread_store_thread; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_thread
    ADD CONSTRAINT uk_store_thread_store_thread UNIQUE (store_id, thread_id);


--
-- Name: app_user_store unique_app_user_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_store
    ADD CONSTRAINT unique_app_user_id UNIQUE (app_user_id);


--
-- Name: product_post unique_columns; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_post
    ADD CONSTRAINT unique_columns UNIQUE (app_user_id, post_id, product_id);


--
-- Name: otp unique_email; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otp
    ADD CONSTRAINT unique_email UNIQUE (email);


--
-- Name: team unique_store_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team
    ADD CONSTRAINT unique_store_id UNIQUE (store_id);


--
-- Name: team_app_user unique_team_app_user; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_app_user
    ADD CONSTRAINT unique_team_app_user UNIQUE (team_id, app_user_id);


--
-- Name: team_channel_account unique_team_id_channel_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_channel_account
    ADD CONSTRAINT unique_team_id_channel_id UNIQUE (team_id, channel_id);


--
-- Name: organization_app_user unique_user_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_app_user
    ADD CONSTRAINT unique_user_id UNIQUE (app_user_id);


--
-- Name: post_score uq_post_score_post_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_score
    ADD CONSTRAINT uq_post_score_post_id UNIQUE (post_id);


--
-- Name: post_word_attribute uq_post_word_attr; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_word_attribute
    ADD CONSTRAINT uq_post_word_attr UNIQUE (post_word_id, attribute, type, attribute_column);


--
-- Name: ai_analysis_log_channel_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ai_analysis_log_channel_idx ON public.ai_analysis_log USING btree (channel);


--
-- Name: ai_analysis_log_group_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ai_analysis_log_group_id_idx ON public.ai_analysis_log USING btree (group_id);


--
-- Name: ai_analysis_log_store_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ai_analysis_log_store_id_idx ON public.ai_analysis_log USING btree (store_id);


--
-- Name: boost_answer_map_code_store_id_boost_question_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_answer_map_code_store_id_boost_question_id_index ON public.boost_answer_map USING btree (code DESC, store_id DESC, boost_question_id DESC);


--
-- Name: boost_post_similarity_store_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_post_similarity_store_id_idx ON public.boost_post_similarity USING btree (store_id);


--
-- Name: boost_present_ledger_order_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_ledger_order_id_idx ON public.boost_present_ledger USING btree (order_id);


--
-- Name: boost_present_ledger_store_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_ledger_store_id_idx ON public.boost_present_ledger USING btree (order_id);


--
-- Name: boost_present_order_attribute_key_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_order_attribute_key_idx ON public.boost_present_order_attribute USING btree (key);


--
-- Name: boost_present_order_attribute_order_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_order_attribute_order_id_idx ON public.boost_present_order_attribute USING btree (order_id);


--
-- Name: boost_present_order_boost_present_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_order_boost_present_id_idx ON public.boost_present_order USING btree (boost_present_id);


--
-- Name: boost_present_order_history_order_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_order_history_order_id_idx ON public.boost_present_order_history USING btree (order_id);


--
-- Name: boost_present_order_history_transaction_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_order_history_transaction_id_idx ON public.boost_present_order_history USING btree (transaction_id);


--
-- Name: boost_present_order_store_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_order_store_id_idx ON public.boost_present_order USING btree (store_id);


--
-- Name: boost_present_order_uuid_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_order_uuid_idx ON public.boost_present_order USING btree (uuid);


--
-- Name: boost_present_store_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_present_store_id_idx ON public.boost_present USING btree (store_id);


--
-- Name: boost_question_map_code_store_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_question_map_code_store_id_index ON public.boost_question_map USING btree (code DESC, store_id DESC);


--
-- Name: boost_receipt_info_store_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_receipt_info_store_id_index ON public.boost_receipt_info USING btree (store_id);


--
-- Name: boost_receipt_payments_message_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_receipt_payments_message_id_index ON public.boost_receipt_payments USING btree (message_id);


--
-- Name: boost_receipt_payments_receipt_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_receipt_payments_receipt_id_index ON public.boost_receipt_payments USING btree (receipt_id);


--
-- Name: boost_receipts_hospital_id_patient_id_checkin_time_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_receipts_hospital_id_patient_id_checkin_time_index ON public.boost_receipts USING btree (hospital_id, patient_id, checkin_time);


--
-- Name: boost_store_info_store_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_store_info_store_id_index ON public.boost_store_info USING btree (store_id DESC);


--
-- Name: boost_treatment_code_code_type_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_treatment_code_code_type_index ON public.boost_treatment_code USING btree (code DESC, type DESC);


--
-- Name: boost_treatment_map_code_store_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX boost_treatment_map_code_store_id_index ON public.boost_treatment_map USING btree (code DESC, store_id DESC);


--
-- Name: channel_analysis_info_channel_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX channel_analysis_info_channel_idx ON public.channel_analysis_info USING btree (channel);


--
-- Name: crawling_log_store_channel_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX crawling_log_store_channel_id_idx ON public.crawling_log USING btree (store_channel_id);


--
-- Name: flyway_schema_history_s_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX flyway_schema_history_s_idx ON public.flyway_schema_history USING btree (success);


--
-- Name: idx_bdpi_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_bdpi_lookup ON public.boost_dental_patient_info USING btree (hospital_id, patient_id, treatment_date, chart_number);


--
-- Name: idx_boost_dental_image_patient_filename; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_boost_dental_image_patient_filename ON public.boost_dental_image USING btree (dental_patient_id, filename);


--
-- Name: idx_hospital_treatment_positive_review_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hospital_treatment_positive_review_post_id ON public.hospital_treatment_positive_review USING btree (post_id);


--
-- Name: idx_hospital_treatment_positive_review_summary_hospital; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hospital_treatment_positive_review_summary_hospital ON public.hospital_treatment_positive_review_summary USING btree (hospital_id);


--
-- Name: idx_post_classification_translation_locale; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_classification_translation_locale ON public.post_classification_translation USING btree (locale);


--
-- Name: idx_post_classification_translation_post_classification_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_classification_translation_post_classification_id ON public.post_classification_translation USING btree (post_classification_id);


--
-- Name: idx_post_reply_ai_recommend_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_reply_ai_recommend_post_id ON public.post_reply_ai_recommend USING btree (post_id);


--
-- Name: idx_post_reply_platform_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_reply_platform_id ON public.post_reply USING btree (platform_id) WHERE (platform_id IS NOT NULL);


--
-- Name: idx_post_reply_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_reply_post_id ON public.post_reply USING btree (post_id);


--
-- Name: idx_post_reply_registration_request_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_reply_registration_request_post_id ON public.post_reply_registration_request USING btree (post_id);


--
-- Name: idx_post_score_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_score_post_id ON public.post_score USING btree (post_id);


--
-- Name: idx_post_store_channel_id_usable; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_store_channel_id_usable ON public.post USING btree (store_channel_id, usable);


--
-- Name: idx_post_treatment_translation_locale; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_treatment_translation_locale ON public.post_treatment_translation USING btree (locale);


--
-- Name: idx_post_treatment_translation_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_treatment_translation_post_id ON public.post_treatment_translation USING btree (post_id);


--
-- Name: idx_post_usable_store_channel_treatment; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_usable_store_channel_treatment ON public.post USING btree (store_channel_id, treatment) WHERE ((usable = true) AND (treatment IS NOT NULL));


--
-- Name: idx_post_word_attribute_word; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_word_attribute_word ON public.post_word_attribute USING btree (post_word_id, attribute, type);


--
-- Name: idx_post_word_post_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_post_word_post_id ON public.post_word USING btree (post_id);


--
-- Name: idx_store_channel_store_id_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_store_channel_store_id_id ON public.store_channel USING btree (store_id, id);


--
-- Name: match_review_analysis_data_ai_analysis_log_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX match_review_analysis_data_ai_analysis_log_id_idx ON public.match_review_analysis_data USING btree (ai_analysis_log_id);


--
-- Name: match_review_analysis_data_post_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX match_review_analysis_data_post_id_idx ON public.match_review_analysis_data USING btree (post_id);


--
-- Name: match_review_analysis_data_thread_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX match_review_analysis_data_thread_id_idx ON public.match_review_analysis_data USING btree (thread_id);


--
-- Name: post_message_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX post_message_id_index ON public.post USING btree (message_id);


--
-- Name: post_treatment_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX post_treatment_index ON public.post USING btree (treatment);


--
-- Name: prompt_template_prompt_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX prompt_template_prompt_type_idx ON public.prompt_template USING btree (prompt_type);


--
-- Name: relevance_analysis_data_ai_analysis_log_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX relevance_analysis_data_ai_analysis_log_id_idx ON public.relevance_analysis_data USING btree (ai_analysis_log_id);


--
-- Name: relevance_analysis_data_post_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX relevance_analysis_data_post_id_idx ON public.relevance_analysis_data USING btree (post_id);


--
-- Name: relevance_analysis_data_thread_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX relevance_analysis_data_thread_id_idx ON public.relevance_analysis_data USING btree (thread_id);


--
-- Name: similarity_score_analysis_data_ai_analysis_log_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX similarity_score_analysis_data_ai_analysis_log_id_idx ON public.similarity_score_analysis_data USING btree (ai_analysis_log_id);


--
-- Name: store_channel_store_id_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX store_channel_store_id_index ON public.store_channel USING btree (store_id);


--
-- Name: store_relevance_info_store_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX store_relevance_info_store_id_idx ON public.store_relevance_info USING btree (store_id);


--
-- Name: store_relevance_keyword_store_relevance_info_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX store_relevance_keyword_store_relevance_info_id_idx ON public.store_relevance_keyword USING btree (store_relevance_info_id, code);


--
-- Name: thread_reply n8n_trigger_0421e4c2_155e_4d22_b5f4_600e4845a5f1; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER n8n_trigger_0421e4c2_155e_4d22_b5f4_600e4845a5f1 AFTER INSERT ON public.thread_reply FOR EACH ROW EXECUTE FUNCTION public.n8n_trigger_function_0421e4c2_155e_4d22_b5f4_600e4845a5f1();


--
-- Name: thread_comment n8n_trigger_25ebbf08_ec3e_4881_9b70_538cc6af5ab8; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER n8n_trigger_25ebbf08_ec3e_4881_9b70_538cc6af5ab8 AFTER INSERT ON public.thread_comment FOR EACH ROW EXECUTE FUNCTION public.n8n_trigger_function_25ebbf08_ec3e_4881_9b70_538cc6af5ab8();


--
-- Name: thread n8n_trigger_ae5e7d23_251d_4612_b4ea_6ca75de169f4; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER n8n_trigger_ae5e7d23_251d_4612_b4ea_6ca75de169f4 AFTER INSERT ON public.thread FOR EACH ROW EXECUTE FUNCTION public.n8n_trigger_function_ae5e7d23_251d_4612_b4ea_6ca75de169f4();


--
-- Name: app_user_attribute FK_app_user_TO_app_user_attribute_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_attribute
    ADD CONSTRAINT "FK_app_user_TO_app_user_attribute_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id);


--
-- Name: app_user_store FK_app_user_TO_app_user_store_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_store
    ADD CONSTRAINT "FK_app_user_TO_app_user_store_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id) ON DELETE CASCADE;


--
-- Name: organization_app_user FK_app_user_TO_organization_app_user_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_app_user
    ADD CONSTRAINT "FK_app_user_TO_organization_app_user_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id);


--
-- Name: team_app_user FK_app_user_TO_organization_app_user_team_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_app_user
    ADD CONSTRAINT "FK_app_user_TO_organization_app_user_team_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id);


--
-- Name: post_reply_registration_request FK_app_user_TO_post_reply_registration_request_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_registration_request
    ADD CONSTRAINT "FK_app_user_TO_post_reply_registration_request_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id);


--
-- Name: product_post FK_app_user_TO_product_post_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_post
    ADD CONSTRAINT "FK_app_user_TO_product_post_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id);


--
-- Name: team_auth FK_app_user_TO_team_auth_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_auth
    ADD CONSTRAINT "FK_app_user_TO_team_auth_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id);


--
-- Name: team_channel_account FK_app_user_TO_team_channel_account_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_channel_account
    ADD CONSTRAINT "FK_app_user_TO_team_channel_account_1" FOREIGN KEY (app_user_id) REFERENCES public.app_user(id);


--
-- Name: boost_customer_gift FK_boost_customer_TO_boost_customer_gift_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_customer_gift
    ADD CONSTRAINT "FK_boost_customer_TO_boost_customer_gift_1" FOREIGN KEY (customer_id) REFERENCES public.boost_customer(id);


--
-- Name: boost_message FK_boost_customer_TO_boost_message_send_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_message
    ADD CONSTRAINT "FK_boost_customer_TO_boost_message_send_1" FOREIGN KEY (customer_id) REFERENCES public.boost_customer(id);


--
-- Name: boost_customer_gift FK_boost_gift_TO_boost_customer_gift_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_customer_gift
    ADD CONSTRAINT "FK_boost_gift_TO_boost_customer_gift_1" FOREIGN KEY (gift_id) REFERENCES public.boost_gift(id);


--
-- Name: boost_team_gift FK_boost_gift_TO_boost_team_gift_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_gift
    ADD CONSTRAINT "FK_boost_gift_TO_boost_team_gift_1" FOREIGN KEY (gift_id) REFERENCES public.boost_gift(id);


--
-- Name: boost_customer_gift FK_boost_message_TO_boost_customer_gift_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_customer_gift
    ADD CONSTRAINT "FK_boost_message_TO_boost_customer_gift_1" FOREIGN KEY (message_id) REFERENCES public.boost_message(id);


--
-- Name: post FK_boost_message_send_TO_post_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post
    ADD CONSTRAINT "FK_boost_message_send_TO_post_1" FOREIGN KEY (message_id) REFERENCES public.boost_message(id);


--
-- Name: boost_message FK_boost_message_template_TO_boost_message_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_message
    ADD CONSTRAINT "FK_boost_message_template_TO_boost_message_1" FOREIGN KEY (message_template_id) REFERENCES public.boost_message_template(id);


--
-- Name: boost_team_message_template FK_boost_message_template_TO_boost_team_message_template_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_message_template
    ADD CONSTRAINT "FK_boost_message_template_TO_boost_team_message_template_1" FOREIGN KEY (message_template_id) REFERENCES public.boost_message_template(id);


--
-- Name: community FK_channel_TO_community_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.community
    ADD CONSTRAINT "FK_channel_TO_community_1" FOREIGN KEY (channel_id) REFERENCES public.channel(id);


--
-- Name: community_keyword FK_community_TO_community_keyword_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.community_keyword
    ADD CONSTRAINT "FK_community_TO_community_keyword_1" FOREIGN KEY (community_id) REFERENCES public.community(id);


--
-- Name: store_community FK_community_TO_store_community_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_community
    ADD CONSTRAINT "FK_community_TO_store_community_1" FOREIGN KEY (community_id) REFERENCES public.community(id);


--
-- Name: store_community_keyword FK_community_TO_store_community_keyword_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_community_keyword
    ADD CONSTRAINT "FK_community_TO_store_community_keyword_1" FOREIGN KEY (community_id) REFERENCES public.community(id);


--
-- Name: thread FK_community_TO_thread_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread
    ADD CONSTRAINT "FK_community_TO_thread_1" FOREIGN KEY (community_id) REFERENCES public.community(id);


--
-- Name: community_keyword FK_keyword_TO_community_keyword_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.community_keyword
    ADD CONSTRAINT "FK_keyword_TO_community_keyword_1" FOREIGN KEY (keyword_id) REFERENCES public.keyword(id);


--
-- Name: store_community_keyword FK_keyword_TO_store_community_keyword_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_community_keyword
    ADD CONSTRAINT "FK_keyword_TO_store_community_keyword_1" FOREIGN KEY (keyword_id) REFERENCES public.keyword(id);


--
-- Name: thread_keyword FK_keyword_TO_thread_keyword_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread_keyword
    ADD CONSTRAINT "FK_keyword_TO_thread_keyword_1" FOREIGN KEY (keyword_id) REFERENCES public.keyword(id);


--
-- Name: organization_app_user FK_organization_TO_organization_app_user_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_app_user
    ADD CONSTRAINT "FK_organization_TO_organization_app_user_1" FOREIGN KEY (organization_id) REFERENCES public.organization(id);


--
-- Name: team FK_organization_TO_team_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team
    ADD CONSTRAINT "FK_organization_TO_team_1" FOREIGN KEY (organization_id) REFERENCES public.organization(id);


--
-- Name: team_app_user FK_organization_TO_team_app_user_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_app_user
    ADD CONSTRAINT "FK_organization_TO_team_app_user_1" FOREIGN KEY (organization_id) REFERENCES public.organization(id);


--
-- Name: team_app_user FK_organization_app_user_TO_organization_app_user_team_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_app_user
    ADD CONSTRAINT "FK_organization_app_user_TO_organization_app_user_team_1" FOREIGN KEY (organization_app_user_id) REFERENCES public.organization_app_user(id);


--
-- Name: boost_customer_gift FK_post_TO_boost_customer_gift_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_customer_gift
    ADD CONSTRAINT "FK_post_TO_boost_customer_gift_1" FOREIGN KEY (post_id) REFERENCES public.post(id);


--
-- Name: post_reply FK_post_TO_post_reply_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply
    ADD CONSTRAINT "FK_post_TO_post_reply_1" FOREIGN KEY (post_id) REFERENCES public.post(id) ON DELETE CASCADE;


--
-- Name: post_reply_ai_recommend FK_post_TO_post_reply_ai_recommend_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_ai_recommend
    ADD CONSTRAINT "FK_post_TO_post_reply_ai_recommend_1" FOREIGN KEY (post_id) REFERENCES public.post(id);


--
-- Name: post_reply_registration_request FK_post_TO_post_reply_registration_request_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_registration_request
    ADD CONSTRAINT "FK_post_TO_post_reply_registration_request_1" FOREIGN KEY (post_id) REFERENCES public.post(id);


--
-- Name: post_word FK_post_TO_post_word_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_word
    ADD CONSTRAINT "FK_post_TO_post_word_1" FOREIGN KEY (post_id) REFERENCES public.post(id) ON DELETE CASCADE;


--
-- Name: product_post FK_product_TO_product_post_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_post
    ADD CONSTRAINT "FK_product_TO_product_post_1" FOREIGN KEY (product_id) REFERENCES public.product(id);


--
-- Name: app_user_store FK_store_TO_app_user_store_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_user_store
    ADD CONSTRAINT "FK_store_TO_app_user_store_1" FOREIGN KEY (store_id) REFERENCES public.store(id) ON DELETE CASCADE;


--
-- Name: store_channel FK_store_TO_store_channel_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_channel
    ADD CONSTRAINT "FK_store_TO_store_channel_1" FOREIGN KEY (store_id) REFERENCES public.store(id) ON DELETE CASCADE;


--
-- Name: store_community FK_store_TO_store_community_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_community
    ADD CONSTRAINT "FK_store_TO_store_community_1" FOREIGN KEY (store_id) REFERENCES public.store(id);


--
-- Name: store_community_keyword FK_store_TO_store_community_keyword_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_community_keyword
    ADD CONSTRAINT "FK_store_TO_store_community_keyword_1" FOREIGN KEY (store_id) REFERENCES public.store(id);


--
-- Name: store_thread FK_store_TO_store_thread_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_thread
    ADD CONSTRAINT "FK_store_TO_store_thread_1" FOREIGN KEY (store_id) REFERENCES public.store(id);


--
-- Name: team FK_store_TO_team_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team
    ADD CONSTRAINT "FK_store_TO_team_1" FOREIGN KEY (store_id) REFERENCES public.store(id);


--
-- Name: team_auth FK_store_TO_team_auth_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_auth
    ADD CONSTRAINT "FK_store_TO_team_auth_1" FOREIGN KEY (store_id) REFERENCES public.store(id);


--
-- Name: post FK_store_channel_TO_post_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post
    ADD CONSTRAINT "FK_store_channel_TO_post_1" FOREIGN KEY (store_channel_id) REFERENCES public.store_channel(id) ON DELETE CASCADE;


--
-- Name: store_channel_attribute FK_store_channel_TO_store_channel_attribute_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_channel_attribute
    ADD CONSTRAINT "FK_store_channel_TO_store_channel_attribute_1" FOREIGN KEY (store_channel) REFERENCES public.store_channel(id);


--
-- Name: boost_customer_gift FK_team_TO_boost_customer_gift_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_customer_gift
    ADD CONSTRAINT "FK_team_TO_boost_customer_gift_1" FOREIGN KEY (team_id) REFERENCES public.team(id);


--
-- Name: boost_message FK_team_TO_boost_message_send_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_message
    ADD CONSTRAINT "FK_team_TO_boost_message_send_1" FOREIGN KEY (team_id) REFERENCES public.team(id);


--
-- Name: boost_team_gift FK_team_TO_boost_team_gift_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_gift
    ADD CONSTRAINT "FK_team_TO_boost_team_gift_1" FOREIGN KEY (team_id) REFERENCES public.team(id);


--
-- Name: boost_team_message_template FK_team_TO_boost_team_message_template_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.boost_team_message_template
    ADD CONSTRAINT "FK_team_TO_boost_team_message_template_1" FOREIGN KEY (team_id) REFERENCES public.team(id);


--
-- Name: team_app_user FK_team_TO_organization_app_user_team_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_app_user
    ADD CONSTRAINT "FK_team_TO_organization_app_user_team_1" FOREIGN KEY (team_id) REFERENCES public.team(id);


--
-- Name: team_channel_account FK_team_TO_team_channel_account_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_channel_account
    ADD CONSTRAINT "FK_team_TO_team_channel_account_1" FOREIGN KEY (team_id) REFERENCES public.team(id);


--
-- Name: store_thread FK_thread_TO_store_thread_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.store_thread
    ADD CONSTRAINT "FK_thread_TO_store_thread_1" FOREIGN KEY (thread_id) REFERENCES public.thread(id);


--
-- Name: thread_comment FK_thread_TO_thread_comment_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread_comment
    ADD CONSTRAINT "FK_thread_TO_thread_comment_1" FOREIGN KEY (thread_id) REFERENCES public.thread(id);


--
-- Name: thread_keyword FK_thread_TO_thread_keyword_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread_keyword
    ADD CONSTRAINT "FK_thread_TO_thread_keyword_1" FOREIGN KEY (thread_id) REFERENCES public.thread(id);


--
-- Name: thread_reply FK_thread_comment_TO_thread_reply_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.thread_reply
    ADD CONSTRAINT "FK_thread_comment_TO_thread_reply_1" FOREIGN KEY (thread_comment_id) REFERENCES public.thread_comment(id);


--
-- Name: post_reply_child fk_post_reply_child_post_reply; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_reply_child
    ADD CONSTRAINT fk_post_reply_child_post_reply FOREIGN KEY (post_reply_id) REFERENCES public.post_reply(id);


--
-- Name: post_word_attribute fk_post_word; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_word_attribute
    ADD CONSTRAINT fk_post_word FOREIGN KEY (post_word_id) REFERENCES public.post_word(id) ON DELETE CASCADE;


--
-- Name: post_classification post_classification_post_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.post_classification
    ADD CONSTRAINT post_classification_post_id_fkey FOREIGN KEY (post_id) REFERENCES public.post(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--


