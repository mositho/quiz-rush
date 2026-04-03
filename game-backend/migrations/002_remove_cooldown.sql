-- Wrong answers now deduct time from ends_at directly instead of entering a
-- cooldown hold state, so these columns are no longer needed.

alter table game_sessions
    drop column if exists cooldown_until,
    drop constraint if exists game_sessions_cooldown_until_check,
    drop constraint if exists game_sessions_status_check,
    add constraint game_sessions_status_check
        check (status in ('active', 'finished'));

alter table game_session_questions
    drop column if exists cooldown_applied_ms;
