create extension if not exists "pgcrypto";

create table if not exists user_profiles (
    id uuid primary key default gen_random_uuid(),
    public_user_id text not null unique,
    keycloak_subject text unique,
    display_name text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists game_sessions (
    id uuid primary key default gen_random_uuid(),
    owner_profile_id uuid references user_profiles(id) on delete set null,
    is_anonymous boolean not null default true,
    status text not null check (status in ('active', 'cooldown', 'finished')),
    finish_reason text check (
        finish_reason in (
            'timer_elapsed',
            'question_pool_exhausted',
            'manual_finish',
            'quit'
        )
    ),
    started_at timestamptz not null default now(),
    ends_at timestamptz not null,
    cooldown_until timestamptz,
    finished_at timestamptz,
    save_deadline_at timestamptz,
    duration_seconds integer not null check (duration_seconds > 0),
    selected_question_set_ids jsonb not null default '[]'::jsonb,
    configuration_key text not null,
    current_question_index integer,
    total_questions integer not null default 0 check (total_questions >= 0),
    answered_questions integer not null default 0 check (answered_questions >= 0),
    correct_questions integer not null default 0 check (correct_questions >= 0),
    wrong_questions integer not null default 0 check (wrong_questions >= 0),
    current_score integer not null default 0 check (current_score >= 0),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    check (jsonb_typeof(selected_question_set_ids) = 'array'),
    check (jsonb_array_length(selected_question_set_ids) > 0),
    check (
        finish_reason is null
        or status = 'finished'
    ),
    check (
        cooldown_until is null
        or status = 'cooldown'
    ),
    check (
        (is_anonymous and owner_profile_id is null)
        or (not is_anonymous and owner_profile_id is not null)
    )
);

create table if not exists game_session_questions (
    id uuid primary key default gen_random_uuid(),
    session_id uuid not null references game_sessions(id) on delete cascade,
    position integer not null check (position >= 0),
    question_id text not null,
    question_set_id text not null,
    difficulty integer not null check (difficulty >= 0),
    question_categories jsonb not null default '[]'::jsonb,
    question_text text not null,
    options_json jsonb not null,
    correct_answer_index integer not null check (correct_answer_index >= 0),
    activated_at timestamptz,
    answered_at timestamptz,
    selected_answer_index integer check (selected_answer_index >= 0),
    is_correct boolean,
    awarded_points integer check (awarded_points >= 0),
    response_time_ms integer check (response_time_ms >= 0),
    cooldown_applied_ms integer check (cooldown_applied_ms >= 0),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (session_id, position),
    check (jsonb_typeof(question_categories) = 'array'),
    check (jsonb_typeof(options_json) = 'array')
);

create table if not exists game_scores (
    id uuid primary key default gen_random_uuid(),
    session_id uuid not null unique references game_sessions(id) on delete restrict,
    owner_profile_id uuid references user_profiles(id) on delete set null,
    is_saved boolean not null default false,
    is_public boolean not null default false,
    created_at timestamptz not null default now(),
    finished_at timestamptz not null,
    saved_at timestamptz,
    expires_at timestamptz,
    finish_reason text not null check (
        finish_reason in (
            'timer_elapsed',
            'question_pool_exhausted',
            'manual_finish',
            'quit'
        )
    ),
    score integer not null check (score >= 0),
    correct_questions integer not null check (correct_questions >= 0),
    wrong_questions integer not null check (wrong_questions >= 0),
    answered_questions integer not null check (answered_questions >= 0),
    total_questions integer not null check (total_questions >= 0),
    duration_seconds integer not null check (duration_seconds > 0),
    played_ms integer not null check (played_ms >= 0),
    selected_question_set_ids jsonb not null default '[]'::jsonb,
    configuration_key text not null,
    question_results_json jsonb not null default '[]'::jsonb,
    check (jsonb_typeof(selected_question_set_ids) = 'array'),
    check (jsonb_array_length(selected_question_set_ids) > 0),
    check (jsonb_typeof(question_results_json) = 'array')
    ,
    check (
        (is_saved and owner_profile_id is not null and saved_at is not null and expires_at is null)
        or (not is_saved and saved_at is null)
    ),
    check (
        (is_public and is_saved and owner_profile_id is not null)
        or not is_public
    ),
    check (
        (owner_profile_id is null and expires_at is not null)
        or (owner_profile_id is not null and expires_at is null)
    )
);

create index if not exists idx_user_profiles_keycloak_subject
    on user_profiles(keycloak_subject);

create index if not exists idx_game_sessions_owner_status
    on game_sessions(owner_profile_id, status);

create index if not exists idx_game_sessions_save_deadline_at
    on game_sessions(save_deadline_at);

create index if not exists idx_game_session_questions_session_position
    on game_session_questions(session_id, position);

create index if not exists idx_game_scores_owner_saved
    on game_scores(owner_profile_id, is_saved);

create index if not exists idx_game_scores_public_saved_score
    on game_scores(is_public, is_saved, score desc);

create index if not exists idx_game_scores_configuration_public_saved_score
    on game_scores(configuration_key, is_public, is_saved, score desc);
