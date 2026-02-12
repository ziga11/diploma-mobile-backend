CREATE OR REPLACE FUNCTION mobile.fetch_message_thread(
    v_main_message_id integer,
    v_user_id integer
)
RETURNS TABLE (
    message_id int,
    sender_id int,
    recipient_id int,
    date timestamptz,
    parent_msg_id int,
    read boolean,
    translations jsonb
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE message_tree AS (
        SELECT
            m.id AS message_id,
            m.creator_id AS sender_id,
            um_other.user_id AS recipient_id,
            m.date::timestamptz AS date,
            m.parent_msg_id,
            COALESCE(um_requesting.read, false) AS read
        FROM mobile.message m
        LEFT JOIN mobile.user_messages um_requesting
            ON um_requesting.message_id = m.id
           AND um_requesting.user_id = v_user_id
        LEFT JOIN mobile.user_messages um_other
            ON um_other.message_id = m.id
           AND um_other.user_id != m.creator_id
        WHERE m.id = v_main_message_id
        
        UNION ALL
        
        SELECT
            m.id,
            m.creator_id,
            um_other.user_id AS recipient_id,
            m.date::timestamptz,
            m.parent_msg_id,
            COALESCE(um_requesting.read, false)
        FROM mobile.message m
        INNER JOIN message_tree mt ON m.parent_msg_id = mt.message_id
        LEFT JOIN mobile.user_messages um_requesting
            ON um_requesting.message_id = m.id 
            AND um_requesting.user_id = v_user_id
        LEFT JOIN mobile.user_messages um_other 
            ON um_other.message_id = m.id 
            AND um_other.user_id != m.creator_id
    )
    SELECT 
        t.message_id,
        t.sender_id,
        t.recipient_id,
        t.date,
        t.parent_msg_id,
        t.read,
        (SELECT jsonb_object_agg(
            mt.language_code,
            jsonb_build_object('title', mt.title, 'body', mt.body))
         FROM mobile.message_translations mt
         WHERE mt.message_id = t.message_id
        ) AS translations
    FROM message_tree t
    ORDER BY t.date;
END;
$$;

CREATE OR REPLACE FUNCTION mobile.fetch_message_root(
    v_message_id int
)
RETURNS TABLE (
    message_id int,
    translations jsonb
)
LANGUAGE sql
AS $$
    WITH RECURSIVE thread_root AS (
        SELECT id, parent_msg_id
        FROM mobile.message
        WHERE id = v_message_id
        
        UNION ALL
        
        SELECT m.id, m.parent_msg_id
        FROM mobile.message m
        JOIN thread_root tr ON m.id = tr.parent_msg_id
    )
    SELECT 
        m.id,
        (SELECT jsonb_object_agg(
            mt.language_code,
            jsonb_build_object('title', mt.title, 'body', mt.body))
         FROM mobile.message_translations mt
         WHERE mt.message_id = m.id
        ) AS translations
    FROM thread_root tr
    JOIN mobile.message m ON m.id = tr.id
    WHERE tr.parent_msg_id IS NULL;
$$;

CREATE OR REPLACE FUNCTION mobile.fetch_root_messages(
    v_user_id integer
)
RETURNS TABLE (
    message_id int,
    sender_id int,
    recipient_id int,
    date timestamptz,
    read boolean,
    translations jsonb
)
LANGUAGE sql
AS $$
    SELECT
        m.id,
        m.creator_id AS sender_id,
        um_other.user_id AS recipient_id,
        m.date,
        COALESCE(um_requesting.read, false) AS read,
        jsonb_object_agg(
            mt.language_code,
            jsonb_build_object(
                'title', mt.title,
                'body', mt.body
            )
        ) AS translations
    FROM mobile.message m
    LEFT JOIN mobile.user_messages um_requesting
        ON um_requesting.message_id = m.id
       AND um_requesting.user_id = v_user_id

    JOIN mobile.user_messages um_creator
        ON um_creator.message_id = m.id
       AND um_creator.user_id = m.creator_id

    JOIN mobile.user_messages um_other
        ON um_other.message_id = m.id
       AND um_other.user_id != m.creator_id

    JOIN mobile.message_translations mt
        ON m.id = mt.message_id
    WHERE 
        m.parent_msg_id IS NULL
        AND (v_user_id = m.creator_id OR v_user_id = um_other.user_id)
    GROUP BY
        m.id,
        m.creator_id,
        um_other.user_id,
        m.date,
        um_requesting.read;
$$;

CREATE OR REPLACE FUNCTION mobile.fetch_user_obligations(p_user_id INTEGER)
RETURNS TABLE (
    id INTEGER,
    user_id INTEGER,
    translations JSONB,
    example_doc_id INTEGER,
    uploadable BOOLEAN,
    text_field BOOLEAN,
    google_file_id TEXT,
    user_document_ids integer[],
    status TEXT,
    reasoning TEXT,
    date TIMESTAMPTZ,
    text_value TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        o.id,
        ub.user_id as user_id,
        jsonb_object_agg(
            ot.language_code,
            jsonb_build_object(
                'title', ot.title,
                'body', COALESCE(ot.body, '')
            )
        ) AS translations,
        o.document_id as example_id,
        o.uploadable,
        o.text_field,
        jed.google_file_id,
        COALESCE(
            array_agg(DISTINCT uod.document_id)
                FILTER (WHERE uod.document_id IS NOT NULL),
            ARRAY[]::integer[]
        ) AS user_document_ids,
        ub.status::text,
        ub.reasoning,
        ub.date,
        COALESCE(ub.text_value, '')
    FROM mobile.obligation o
    JOIN mobile.user_obligations ub ON ub.obligation_id = o.id
    LEFT JOIN mobile.obligation_translations ot
        ON ot.obligation_id = o.id
    LEFT JOIN mobile.user_obligation_documents uod
        ON uod.link_id = ub.id
    LEFT JOIN mobile.user_job uj
        ON uj.user_id = ub.user_id
    LEFT JOIN mobile.job_external_documents jed
        ON jed.obligation_id = o.id AND
           jed.job_id = uj.job_id
    WHERE
        ub.user_id = p_user_id
    GROUP BY
        o.id,
        ub.user_id,
        o.document_id,
        o.uploadable,
        o.text_field,
        jed.google_file_id,
        ub.status,
        ub.reasoning,
        ub.date,
        ub.text_value;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION mobile.fetch_notifications(
    v_user_id integer
)
RETURNS TABLE (
    link_id int,
    read boolean,
    date timestamptz,
    type text,
    suitable boolean,
    thread_ts text,
    translations jsonb
)
LANGUAGE sql
AS $$
	SELECT 
		un.link_id,
		un.read,
		un.date,
		n.type,
		un.suitable,
		un.thread_ts,
		jsonb_object_agg(
		    nt.language_code,
		    jsonb_build_object(
			'title', nt.title,
			'body', nt.body
		     )
		) AS translations
	FROM mobile.user_notifications un 
	JOIN mobile.notification n
		ON n.ID = un.notification_id
	JOIN mobile.notification_translations nt
		ON nt.notification_id = n.id
	JOIN mobile.users u ON
		un.user_id = u.id
	WHERE 
		user_id = v_user_id
	GROUP BY
		un.link_id,
		un.read,
		un.date,
		n.type,
		un.suitable,
		un.thread_ts
	ORDER BY
		un.date DESC
$$;


CREATE OR REPLACE FUNCTION projektni_obrazec.upsert_company(project_title TEXT)
RETURNS INT
LANGUAGE plpgsql
AS $$
DECLARE
    v_company_id INT;
BEGIN
    INSERT INTO mobile.company (name)
    	VALUES (project_title)
    	ON CONFLICT (name)
	    DO UPDATE
		SET name = EXCLUDED.name
    RETURNING id INTO v_company_id;

    RETURN v_company_id;
END;
$$;

CREATE OR REPLACE FUNCTION projektni_obrazec.create_jobs_for_company(
    p_company_id INT,
    p_jobs_json JSONB
)
RETURNS INT[]  
LANGUAGE plpgsql
AS $$
DECLARE
    v_job_ids INT[];
BEGIN
    WITH inserted_jobs AS (
        INSERT INTO mobile.job (company_id, title)
        SELECT 
            p_company_id,
            j.value->>'value'
        FROM jsonb_each(p_jobs_json) AS j(key, value)
        WHERE j.key ILIKE '%ime_delovnega_mesta%'
	ON CONFLICT (title, company_id)
	    DO UPDATE
		SET title = EXCLUDED.title
        RETURNING id
    )
    SELECT ARRAY(SELECT id FROM inserted_jobs) INTO v_job_ids;
    
    RETURN COALESCE(v_job_ids, ARRAY[]::INT[]);
END;
$$;

CREATE OR REPLACE FUNCTION projektni_obrazec.insert_obligation_applicability(
    p_job_ids INT[],
    p_json JSONB
)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM mobile.obligation_applicability
    WHERE job_id = ANY(p_job_ids);

    INSERT INTO mobile.obligation_applicability(job_id, obligation_id, workpermit_status)
        WITH field_obligation AS (
            SELECT DISTINCT fom.obligation_id
            FROM projektni_obrazec.field_obligation_mapping fom
            WHERE (p_json #> fom.full_path ->> 'value')::boolean = true
        ),
        unique_jobs AS (
            SELECT DISTINCT unnest(p_job_ids) AS job_id
        )
        SELECT 
            j.job_id,
            fo.obligation_id,
            status_list.status
        FROM field_obligation fo
        JOIN unique_jobs j ON true
        JOIN (VALUES ('HAS'), ('EU_EFTA'), ('TEMPORARY')) AS status_list(status) ON true;
END;
$$;

CREATE OR REPLACE FUNCTION projektni_obrazec.insert_job_external_documents(
    p_json JSONB,
    p_job_ids INT[]
)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO mobile.job_external_documents(job_id, obligation_id, google_file_id)
    WITH document_urls AS (
        SELECT
            fom.obligation_id,
            (p_json #> fom.full_path -> 'children' -> fom.field_document_name ->> 'value') AS google_file_id
        FROM projektni_obrazec.field_obligation_mapping fom
        WHERE 
            fom.field_document_name IS NOT NULL
            AND fom.field_document_name != ''
    )
    SELECT
        unnest(p_job_ids),
        obligation_id,
        google_file_id
    FROM document_urls
    WHERE google_file_id IS NOT NULL 
      AND google_file_id != ''
    ON CONFLICT (obligation_id, job_id) 
    DO UPDATE SET 
        google_file_id = EXCLUDED.google_file_id;
END;
$$;

CREATE OR REPLACE FUNCTION projektni_obrazec.on_project_change()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    v_company_id INT;
    v_jobs_json JSONB;
    v_job_ids INT[];
BEGIN
    INSERT INTO projektni_obrazec.debug_log(message)
    VALUES ('Trigger fired for project ID: ' || NEW.id || ', TG_OP: ' || TG_OP);

    v_company_id := projektni_obrazec.upsert_company(NEW.title);
    v_jobs_json := NEW.json->'ocena_tveganja_za_delovno_mesto';
    v_job_ids := projektni_obrazec.create_jobs_for_company(v_company_id, v_jobs_json);

    INSERT INTO projektni_obrazec.debug_log(message)
    VALUES ('About to insert obligations for job_ids: ' || v_job_ids::TEXT);

    PERFORM projektni_obrazec.insert_obligation_applicability(v_job_ids, NEW.json);
    PERFORM projektni_obrazec.insert_job_external_documents(NEW.json, v_job_ids);

    RETURN NEW;
END;
$function$


CREATE TRIGGER project_insert_or_update
    AFTER INSERT OR UPDATE
	ON projektni_obrazec.project
	FOR EACH ROW
	    EXECUTE FUNCTION projektni_obrazec.on_project_change();



CREATE TABLE projektni_obrazec.field_obligation_mapping (
    field_name text PRIMARY KEY,
    field_document_name text,
    obligation_id integer REFERENCES mobile.obligation(id),
);

INSERT INTO projektni_obrazec.field_obligation_mapping(field_name, field_document_name, obligation_id) VALUES
    ('potrdilo_o_podaljsanju_vize_ali_dd', '', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Dovoljenje za prebivanje – podaljšanje ali prva izdaja')),
    ('rojstni_list_otroka', '', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Rojstni list otroka')),
    ('soglasje_za_uporabo_telefonske_stevilke', 'soglasje_za_uporabo_telefonske_stevilke_dokument', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Soglasje za uporabo telefonske številke')),
    ('kolektivno_zavarovanje', 'kolektivno_zavarovanje_dokument', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Kolektivno zavarovanje')),

    ('ocena_poskusnega_dela', '', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Ocena poskusnega dela')),
    ('napotitev_na_delo', '', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Napotitev na delo')),
    ('izjavo_o_glavnem_delodajalcu', 'ne_zaposli_1_mesec_datoteka', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Izjava o glavnem delodajalcu (če se ne zaposli v 1. mesecu)')),
    ('izjavo_o_prevozu', '', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Izjava o prevozu')),

    ('velikost_delovne_opreme', '', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Velikost delovne opreme')),
    ('kdaj_lahko_zacne', '', (SELECT obligation_id FROM mobile.obligation_translations WHERE title = 'Kdaj lahko začne'));



CREATE OR REPLACE FUNCTION mobile.fetch_messages(v_user_id integer, v_message_id integer DEFAULT NULL::integer)
 RETURNS TABLE(message_id integer, sender_id integer, recipient_id integer, date timestamp with time zone, read boolean, translations jsonb)
 LANGUAGE sql
AS $function$
    SELECT
        m.id,
        m.creator_id AS sender_id,
        um_other.user_id AS recipient_id,
        m.date,
        COALESCE(um_requesting.read, false) AS read,
        jsonb_object_agg(
            mt.language_code,
            jsonb_build_object(
                'title', mt.title,
                'body', mt.body
            )
        ) AS translations
    FROM mobile.message m
    JOIN mobile.user_messages um_requesting
        ON um_requesting.message_id = m.id
       AND um_requesting.user_id = v_user_id
    JOIN mobile.user_messages um_creator
        ON um_creator.message_id = m.id
       AND um_creator.user_id = m.creator_id
    JOIN mobile.user_messages um_other
        ON um_other.message_id = m.id
       AND um_other.user_id != m.creator_id
    JOIN mobile.message_translations mt
        ON m.id = mt.message_id
    WHERE
        (v_message_id IS NULL AND m.parent_msg_id IS NULL)
        OR (v_message_id IS NOT NULL AND m.id = v_message_id)
    AND (v_user_id = m.creator_id OR v_user_id = um_other.user_id)
    GROUP BY
        m.id,
        m.creator_id,
        um_other.user_id,
        m.date,
        um_requesting.read
    ORDER BY m.date DESC;
$function$

