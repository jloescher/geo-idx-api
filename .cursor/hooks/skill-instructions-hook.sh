#!/bin/bash
# Prompt-aware SummonAIKit skill guidance and production contract verification.

if [ "${SUMMONAIKIT_INTERNAL_GENERATION:-}" = "1" ]; then
  exit 0
fi

INPUT="$(cat)"
TEXT="$(printf '%s' "$INPUT" | tr '[:upper:]' '[:lower:]')"
HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
CONTRACT_PATH="$HOOK_DIR/.summonaikit-task-contract.txt"
LIKELY_SKILLS=""
RISK_SIGNALS=""
EXPERIENCE_SIGNALS=""
PROVIDER_DOCS=""
EXPLICIT_OVERRIDE=""
PREFERRED_SERVICES=""

add_skill() {
  case " $LIKELY_SKILLS " in
    *" $1 "*) ;;
    *) LIKELY_SKILLS="$LIKELY_SKILLS $1" ;;
  esac
}

add_signal() {
  case " $RISK_SIGNALS " in
    *" $1 "*) ;;
    *) RISK_SIGNALS="$RISK_SIGNALS
- $1" ;;
  esac
}

add_experience_signal() {
  case " $EXPERIENCE_SIGNALS " in
    *" $1 "*) ;;
    *) EXPERIENCE_SIGNALS="$EXPERIENCE_SIGNALS
- $1" ;;
  esac
}

add_provider_docs() {
  case " $PROVIDER_DOCS " in
    *" $1 "*) ;;
    *) PROVIDER_DOCS="$PROVIDER_DOCS
- $1" ;;
  esac
}

add_preferred_service() {
  case " $PREFERRED_SERVICES " in
    *" $1 "*) ;;
    *) PREFERRED_SERVICES="$PREFERRED_SERVICES
- $1" ;;
  esac
}

run_contract_check() {
  [ -f "$CONTRACT_PATH" ] || exit 0
  CONTRACT="$(cat "$CONTRACT_PATH")"
  HAS_PROD_CONTRACT=""
  HAS_UI_CONTRACT=""
  HAS_RATE_LIMIT_CONTRACT=""
  HAS_WEBHOOK_CONTRACT=""
  HAS_DB_CONCURRENCY_CONTRACT=""
  if printf '%s' "$CONTRACT" | grep -Eiq '^risk:|abuse/rate-limit guard|database/concurrency|webhook/side-effect flow|email/external side effect|API/auth flow'; then HAS_PROD_CONTRACT="1"; fi
  if printf '%s' "$CONTRACT" | grep -Eiq 'abuse/rate-limit guard|rate-limit-abuse'; then HAS_RATE_LIMIT_CONTRACT="1"; fi
  if printf '%s' "$CONTRACT" | grep -Eiq 'webhook/side-effect flow|webhook-side-effect'; then HAS_WEBHOOK_CONTRACT="1"; fi
  if printf '%s' "$CONTRACT" | grep -Eiq 'database/concurrency|database-concurrency'; then HAS_DB_CONCURRENCY_CONTRACT="1"; fi
  if printf '%s' "$CONTRACT" | grep -Eiq 'experience:|UI/interface change|form/flow UX|state coverage|accessibility|responsive layout'; then HAS_UI_CONTRACT="1"; fi
  [ -n "$HAS_PROD_CONTRACT$HAS_UI_CONTRACT" ] || exit 0

  DIFF_FILES="$(git diff --name-only --diff-filter=ACMRT -- . 2>/dev/null)"
  [ -n "$DIFF_FILES" ] || exit 0
  DIFF="$(git diff -- . 2>/dev/null)"
  [ -n "$DIFF" ] || exit 0

  FAILED=""
  if [ -n "$HAS_RATE_LIMIT_CONTRACT" ] && printf '%s' "$CONTRACT" | grep -Eiq 'Cloudflare Workers' && ! printf '%s' "$CONTRACT" | grep -Eiq '^override:1'; then
    if ! printf '%s' "$DIFF" | grep -Eiq 'RateLimit|rate_limits|limit[[:space:]]*\([[:space:]]*\{[[:space:]]*key'; then
      FAILED="$FAILED
- Native provider limiter missing: this repo appears to use Cloudflare Workers; use the provider RateLimit binding through infra/runtime config, or state an explicit user/project override."
    fi
  fi

  if [ -n "$HAS_WEBHOOK_CONTRACT" ] && printf '%s' "$DIFF" | grep -Eiq 'webhook|validateEvent|WebhookVerificationError|POLAR_WEBHOOK_SECRET|order\.created|order\.paid'; then
    if ! printf '%s' "$DIFF" | grep -Eiq 'validateEvent|WebhookVerificationError|signature|401|fail[ -]?closed'; then
      FAILED="$FAILED
- Webhook signature check missing: verify the provider signature before side effects and fail closed when the secret is absent."
    fi
    if printf '%s' "$DIFF" | grep -Eiq 'delete[[:space:]]*\([^)]*(polarWebhookEvent|webhook).*|compensating delete|claim.*delete'; then
      FAILED="$FAILED
- Retry/idempotency warning: avoid fallible compensating deletes for claimed webhook events; prefer transaction rollback or explicit pending/processed status."
    fi
    if printf '%s' "$DIFF" | grep -Eiq 'purchasedAt' &&
       printf '%s' "$DIFF" | grep -Eiq '(select|find|query).{0,240}purchasedAt.{0,240}(update|set)' &&
       ! printf '%s' "$DIFF" | grep -Eiq 'isNull|IS NULL|where.{0,120}purchasedAt|returning'; then
      FAILED="$FAILED
- Business idempotency warning: payment conversion gates should use an atomic conditional write (for example WHERE purchased_at IS NULL RETURNING), not read-then-update."
    fi
  fi

  if [ -n "$HAS_DB_CONCURRENCY_CONTRACT" ] && printf '%s' "$DIFF" | grep -Eiq 'organization|membership|member' &&
     printf '%s' "$DIFF" | grep -Eiq '(insert|create).{0,120}(organization|membership|member)' &&
     ! printf '%s' "$DIFF" | grep -Eiq 'unique index|unique constraint|onConflict|on conflict|do nothing|upsert'; then
    FAILED="$FAILED
- Relational uniqueness warning: membership/link creation needs a DB uniqueness invariant plus idempotent insert/upsert behavior."
  fi

  if [ -n "$HAS_PROD_CONTRACT" ] && printf '%s' "$DIFF" | grep -Eiq 'new[[:space:]]+Map|globalThis|frontend-only|client-side only|setTimeout[[:space:]]*\(|fire-and-forget|process\.env'; then
    FAILED="$FAILED
- Serverless anti-pattern detected in the diff: avoid module/global counters, frontend-only guards, detached timers, and untyped process.env runtime access."
  fi

  if [ -n "$HAS_PROD_CONTRACT" ] && printf '%s' "$DIFF" | grep -Eiq '(select|find|count).{0,220}(insert|update|increment|count[[:space:]]*\+).{0,220}(rate|limit|throttl|counter|otp|abuse)' &&
     ! printf '%s' "$DIFF" | grep -Eiq 'on conflict|upsert|unique constraint|for update|serializable|transaction|atomic|RateLimit|provider primitive|native primitive'; then
    FAILED="$FAILED
- Non-atomic fallback counter detected: DB limiters must use an atomic statement, transaction/lock, unique-window upsert, or provider-native primitive."
  fi

  if [ -n "$HAS_UI_CONTRACT" ]; then
    UI_FILES="$(printf '%s
' "$DIFF_FILES" | grep -E '\.(tsx|jsx|css|scss|sass|mdx)$|(^|/)components/|(^|/)app/|(^|/)pages/' || true)"
    if [ -n "$UI_FILES" ]; then
      UI_DIFF="$(printf '%s
' "$UI_FILES" | xargs git diff -- 2>/dev/null)"
      if printf '%s' "$UI_DIFF" | grep -Eiq 'onSubmit|mutate\(|useMutation|fetch\(|form' &&
         ! printf '%s' "$UI_DIFF" | grep -Eiq 'loading|pending|disabled|isSubmitting|error|retry|toast|aria-|label'; then
        FAILED="$FAILED
- UI/UX state coverage warning: changed form or async UI without obvious loading/disabled/error/recovery handling."
      fi
      if printf '%s' "$UI_DIFF" | grep -Eiq '<(button|input|select|textarea)|role=' &&
         ! printf '%s' "$UI_DIFF" | grep -Eiq 'aria-|label|focus-visible|disabled|type='; then
        FAILED="$FAILED
- Accessibility warning: changed interactive UI without obvious labels, semantics, focus, or disabled handling."
      fi
      if printf '%s' "$UI_DIFF" | grep -Eiq 'className=' &&
         ! printf '%s' "$UI_DIFF" | grep -Eiq 'sm:|md:|lg:|xl:|max-w|min-w|truncate|overflow-|flex-wrap|grid-cols'; then
        FAILED="$FAILED
- Responsive/layout warning: changed styled UI without obvious responsive constraints, truncation, or overflow handling."
      fi
    fi
  fi

  if [ -n "$FAILED" ]; then
    cat <<CHECK_EOF
SUMMONAIKIT CONTRACT CHECK WARNING
$FAILED

Review before finishing if these findings apply to your current changes. This Stop hook is advisory and will not block completion because repository-wide diffs may include unrelated pre-existing work.
CHECK_EOF
    exit 0
  fi

  exit 0
}

if [ "$SUMMONAIKIT_HOOK_PHASE" = "verify" ]; then
  run_contract_check
fi

if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])go([^[:alnum:]_]|$)'; then add_skill 'go'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])fiber([^[:alnum:]_]|$)'; then add_skill 'fiber'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])postgres([^[:alnum:]_]|$)'; then add_skill 'postgres'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])postgresql([^[:alnum:]_]|$)'; then add_skill 'postgresql'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])docker([^[:alnum:]_]|$)'; then add_skill 'docker'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])frontend-design([^[:alnum:]_]|$)|(^|[^[:alnum:]_])frontend[[:space:]]+design([^[:alnum:]_]|$)'; then add_skill 'frontend-design'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])ux([^[:alnum:]_]|$)'; then add_skill 'ux'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])deploy-coolify([^[:alnum:]_]|$)|(^|[^[:alnum:]_])deploy[[:space:]]+coolify([^[:alnum:]_]|$)'; then add_skill 'deploy-coolify'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])deploy-docker([^[:alnum:]_]|$)|(^|[^[:alnum:]_])deploy[[:space:]]+docker([^[:alnum:]_]|$)'; then add_skill 'deploy-docker'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])hosting-coolify([^[:alnum:]_]|$)|(^|[^[:alnum:]_])hosting[[:space:]]+coolify([^[:alnum:]_]|$)'; then add_skill 'hosting-coolify'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])deploy-patroni([^[:alnum:]_]|$)|(^|[^[:alnum:]_])deploy[[:space:]]+patroni([^[:alnum:]_]|$)'; then add_skill 'deploy-patroni'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])hosting-tailscale([^[:alnum:]_]|$)|(^|[^[:alnum:]_])hosting[[:space:]]+tailscale([^[:alnum:]_]|$)'; then add_skill 'hosting-tailscale'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])storage-s3([^[:alnum:]_]|$)|(^|[^[:alnum:]_])storage[[:space:]]+s3([^[:alnum:]_]|$)'; then add_skill 'storage-s3'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])queue-postgresql([^[:alnum:]_]|$)|(^|[^[:alnum:]_])queue[[:space:]]+postgresql([^[:alnum:]_]|$)'; then add_skill 'queue-postgresql'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])auth-api-token([^[:alnum:]_]|$)|(^|[^[:alnum:]_])auth[[:space:]]+api[[:space:]]+token([^[:alnum:]_]|$)'; then add_skill 'auth-api-token'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])cache-postgres([^[:alnum:]_]|$)|(^|[^[:alnum:]_])cache[[:space:]]+postgres([^[:alnum:]_]|$)'; then add_skill 'cache-postgres'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])proxy-web([^[:alnum:]_]|$)|(^|[^[:alnum:]_])proxy[[:space:]]+web([^[:alnum:]_]|$)'; then add_skill 'proxy-web'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])geospatial([^[:alnum:]_]|$)'; then add_skill 'geospatial'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])auth-domain([^[:alnum:]_]|$)|(^|[^[:alnum:]_])auth[[:space:]]+domain([^[:alnum:]_]|$)'; then add_skill 'auth-domain'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])cron([^[:alnum:]_]|$)'; then add_skill 'cron'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])scoping-feature-work([^[:alnum:]_]|$)|(^|[^[:alnum:]_])scoping[[:space:]]+feature[[:space:]]+work([^[:alnum:]_]|$)'; then add_skill 'scoping-feature-work'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])prioritizing-roadmap-bets([^[:alnum:]_]|$)|(^|[^[:alnum:]_])prioritizing[[:space:]]+roadmap[[:space:]]+bets([^[:alnum:]_]|$)'; then add_skill 'prioritizing-roadmap-bets'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])mapping-user-journeys([^[:alnum:]_]|$)|(^|[^[:alnum:]_])mapping[[:space:]]+user[[:space:]]+journeys([^[:alnum:]_]|$)'; then add_skill 'mapping-user-journeys'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])designing-onboarding-paths([^[:alnum:]_]|$)|(^|[^[:alnum:]_])designing[[:space:]]+onboarding[[:space:]]+paths([^[:alnum:]_]|$)'; then add_skill 'designing-onboarding-paths'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])improving-activation-flow([^[:alnum:]_]|$)|(^|[^[:alnum:]_])improving[[:space:]]+activation[[:space:]]+flow([^[:alnum:]_]|$)'; then add_skill 'improving-activation-flow'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])crafting-empty-states([^[:alnum:]_]|$)|(^|[^[:alnum:]_])crafting[[:space:]]+empty[[:space:]]+states([^[:alnum:]_]|$)'; then add_skill 'crafting-empty-states'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])orchestrating-feature-adoption([^[:alnum:]_]|$)|(^|[^[:alnum:]_])orchestrating[[:space:]]+feature[[:space:]]+adoption([^[:alnum:]_]|$)'; then add_skill 'orchestrating-feature-adoption'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])designing-inapp-guidance([^[:alnum:]_]|$)|(^|[^[:alnum:]_])designing[[:space:]]+inapp[[:space:]]+guidance([^[:alnum:]_]|$)'; then add_skill 'designing-inapp-guidance'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])instrumenting-product-metrics([^[:alnum:]_]|$)|(^|[^[:alnum:]_])instrumenting[[:space:]]+product[[:space:]]+metrics([^[:alnum:]_]|$)'; then add_skill 'instrumenting-product-metrics'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])running-product-experiments([^[:alnum:]_]|$)|(^|[^[:alnum:]_])running[[:space:]]+product[[:space:]]+experiments([^[:alnum:]_]|$)'; then add_skill 'running-product-experiments'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])triaging-user-feedback([^[:alnum:]_]|$)|(^|[^[:alnum:]_])triaging[[:space:]]+user[[:space:]]+feedback([^[:alnum:]_]|$)'; then add_skill 'triaging-user-feedback'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])writing-release-notes([^[:alnum:]_]|$)|(^|[^[:alnum:]_])writing[[:space:]]+release[[:space:]]+notes([^[:alnum:]_]|$)'; then add_skill 'writing-release-notes'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])clarifying-market-fit([^[:alnum:]_]|$)|(^|[^[:alnum:]_])clarifying[[:space:]]+market[[:space:]]+fit([^[:alnum:]_]|$)'; then add_skill 'clarifying-market-fit'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])structuring-offer-ladders([^[:alnum:]_]|$)|(^|[^[:alnum:]_])structuring[[:space:]]+offer[[:space:]]+ladders([^[:alnum:]_]|$)'; then add_skill 'structuring-offer-ladders'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])framing-release-stories([^[:alnum:]_]|$)|(^|[^[:alnum:]_])framing[[:space:]]+release[[:space:]]+stories([^[:alnum:]_]|$)'; then add_skill 'framing-release-stories'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])generating-growth-hypotheses([^[:alnum:]_]|$)|(^|[^[:alnum:]_])generating[[:space:]]+growth[[:space:]]+hypotheses([^[:alnum:]_]|$)'; then add_skill 'generating-growth-hypotheses'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])embedding-decision-cues([^[:alnum:]_]|$)|(^|[^[:alnum:]_])embedding[[:space:]]+decision[[:space:]]+cues([^[:alnum:]_]|$)'; then add_skill 'embedding-decision-cues'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])crafting-page-messaging([^[:alnum:]_]|$)|(^|[^[:alnum:]_])crafting[[:space:]]+page[[:space:]]+messaging([^[:alnum:]_]|$)'; then add_skill 'crafting-page-messaging'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])tightening-brand-voice([^[:alnum:]_]|$)|(^|[^[:alnum:]_])tightening[[:space:]]+brand[[:space:]]+voice([^[:alnum:]_]|$)'; then add_skill 'tightening-brand-voice'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])designing-lifecycle-messages([^[:alnum:]_]|$)|(^|[^[:alnum:]_])designing[[:space:]]+lifecycle[[:space:]]+messages([^[:alnum:]_]|$)'; then add_skill 'designing-lifecycle-messages'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])planning-editorial-arcs([^[:alnum:]_]|$)|(^|[^[:alnum:]_])planning[[:space:]]+editorial[[:space:]]+arcs([^[:alnum:]_]|$)'; then add_skill 'planning-editorial-arcs'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])orchestrating-social-rhythm([^[:alnum:]_]|$)|(^|[^[:alnum:]_])orchestrating[[:space:]]+social[[:space:]]+rhythm([^[:alnum:]_]|$)'; then add_skill 'orchestrating-social-rhythm'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])tuning-landing-journeys([^[:alnum:]_]|$)|(^|[^[:alnum:]_])tuning[[:space:]]+landing[[:space:]]+journeys([^[:alnum:]_]|$)'; then add_skill 'tuning-landing-journeys'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])streamlining-signup-steps([^[:alnum:]_]|$)|(^|[^[:alnum:]_])streamlining[[:space:]]+signup[[:space:]]+steps([^[:alnum:]_]|$)'; then add_skill 'streamlining-signup-steps'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])accelerating-first-run([^[:alnum:]_]|$)|(^|[^[:alnum:]_])accelerating[[:space:]]+first[[:space:]]+run([^[:alnum:]_]|$)'; then add_skill 'accelerating-first-run'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])reducing-form-falloff([^[:alnum:]_]|$)|(^|[^[:alnum:]_])reducing[[:space:]]+form[[:space:]]+falloff([^[:alnum:]_]|$)'; then add_skill 'reducing-form-falloff'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])refining-prompt-surfaces([^[:alnum:]_]|$)|(^|[^[:alnum:]_])refining[[:space:]]+prompt[[:space:]]+surfaces([^[:alnum:]_]|$)'; then add_skill 'refining-prompt-surfaces'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])strengthening-upgrade-moments([^[:alnum:]_]|$)|(^|[^[:alnum:]_])strengthening[[:space:]]+upgrade[[:space:]]+moments([^[:alnum:]_]|$)'; then add_skill 'strengthening-upgrade-moments'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])mapping-conversion-events([^[:alnum:]_]|$)|(^|[^[:alnum:]_])mapping[[:space:]]+conversion[[:space:]]+events([^[:alnum:]_]|$)'; then add_skill 'mapping-conversion-events'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])designing-variation-tests([^[:alnum:]_]|$)|(^|[^[:alnum:]_])designing[[:space:]]+variation[[:space:]]+tests([^[:alnum:]_]|$)'; then add_skill 'designing-variation-tests'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])calibrating-paid-campaigns([^[:alnum:]_]|$)|(^|[^[:alnum:]_])calibrating[[:space:]]+paid[[:space:]]+campaigns([^[:alnum:]_]|$)'; then add_skill 'calibrating-paid-campaigns'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])building-acquisition-tools([^[:alnum:]_]|$)|(^|[^[:alnum:]_])building[[:space:]]+acquisition[[:space:]]+tools([^[:alnum:]_]|$)'; then add_skill 'building-acquisition-tools'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])engineering-referral-loops([^[:alnum:]_]|$)|(^|[^[:alnum:]_])engineering[[:space:]]+referral[[:space:]]+loops([^[:alnum:]_]|$)'; then add_skill 'engineering-referral-loops'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])inspecting-search-coverage([^[:alnum:]_]|$)|(^|[^[:alnum:]_])inspecting[[:space:]]+search[[:space:]]+coverage([^[:alnum:]_]|$)'; then add_skill 'inspecting-search-coverage'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])scaling-template-pages([^[:alnum:]_]|$)|(^|[^[:alnum:]_])scaling[[:space:]]+template[[:space:]]+pages([^[:alnum:]_]|$)'; then add_skill 'scaling-template-pages'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])adding-structured-signals([^[:alnum:]_]|$)|(^|[^[:alnum:]_])adding[[:space:]]+structured[[:space:]]+signals([^[:alnum:]_]|$)'; then add_skill 'adding-structured-signals'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])building-compare-hubs([^[:alnum:]_]|$)|(^|[^[:alnum:]_])building[[:space:]]+compare[[:space:]]+hubs([^[:alnum:]_]|$)'; then add_skill 'building-compare-hubs'; fi

if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])rate[[:space:]]+limit([^[:alnum:]_]|$)|(^|[^[:alnum:]_])ratelimit([^[:alnum:]_]|$)|(^|[^[:alnum:]_])too[[:space:]]+many([^[:alnum:]_]|$)|(^|[^[:alnum:]_])throttle([^[:alnum:]_]|$)|(^|[^[:alnum:]_])repeated[[:space:]]+request([^[:alnum:]_]|$)|(^|[^[:alnum:]_])spam([^[:alnum:]_]|$)|(^|[^[:alnum:]_])abuse([^[:alnum:]_]|$)|(^|[^[:alnum:]_])brute[[:space:]]+force([^[:alnum:]_]|$)|(^|[^[:alnum:]_])bruteforce([^[:alnum:]_]|$)|(^|[^[:alnum:]_])otp([^[:alnum:]_]|$)|(^|[^[:alnum:]_])recovery([^[:alnum:]_]|$)'; then add_signal 'abuse/rate-limit guard'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])background([^[:alnum:]_]|$)|(^|[^[:alnum:]_])after[[:space:]]+response([^[:alnum:]_]|$)|(^|[^[:alnum:]_])fire[[:space:]]+and[[:space:]]+forget([^[:alnum:]_]|$)|(^|[^[:alnum:]_])fire-and-forget([^[:alnum:]_]|$)|(^|[^[:alnum:]_])async[[:space:]]+cleanup([^[:alnum:]_]|$)|(^|[^[:alnum:]_])defer([^[:alnum:]_]|$)|(^|[^[:alnum:]_])later([^[:alnum:]_]|$)'; then add_signal 'background/lifecycle work'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])cron([^[:alnum:]_]|$)|(^|[^[:alnum:]_])scheduled([^[:alnum:]_]|$)|(^|[^[:alnum:]_])schedule([^[:alnum:]_]|$)|(^|[^[:alnum:]_])nightly([^[:alnum:]_]|$)|(^|[^[:alnum:]_])daily([^[:alnum:]_]|$)|(^|[^[:alnum:]_])hourly([^[:alnum:]_]|$)|(^|[^[:alnum:]_])recurring([^[:alnum:]_]|$)|(^|[^[:alnum:]_])interval([^[:alnum:]_]|$)'; then add_signal 'scheduled/recurring work'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])cache([^[:alnum:]_]|$)|(^|[^[:alnum:]_])shared[[:space:]]+state([^[:alnum:]_]|$)|(^|[^[:alnum:]_])counter([^[:alnum:]_]|$)|(^|[^[:alnum:]_])lock([^[:alnum:]_]|$)|(^|[^[:alnum:]_])queue([^[:alnum:]_]|$)|(^|[^[:alnum:]_])mutex([^[:alnum:]_]|$)|(^|[^[:alnum:]_])global([^[:alnum:]_]|$)|(^|[^[:alnum:]_])in[[:space:]]+memory([^[:alnum:]_]|$)|(^|[^[:alnum:]_])in-memory([^[:alnum:]_]|$)'; then add_signal 'cache/shared state'; add_skill 'postgres'; add_skill 'postgresql'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])secret([^[:alnum:]_]|$)|(^|[^[:alnum:]_])env([^[:alnum:]_]|$)|(^|[^[:alnum:]_])environment[[:space:]]+variable([^[:alnum:]_]|$)|(^|[^[:alnum:]_])credential([^[:alnum:]_]|$)|(^|[^[:alnum:]_])token([^[:alnum:]_]|$)|(^|[^[:alnum:]_])api[[:space:]]+key([^[:alnum:]_]|$)|(^|[^[:alnum:]_])binding([^[:alnum:]_]|$)'; then add_signal 'secrets/env wiring'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])database([^[:alnum:]_]|$)|(^|[^[:alnum:]_])postgres([^[:alnum:]_]|$)|(^|[^[:alnum:]_])transaction([^[:alnum:]_]|$)|(^|[^[:alnum:]_])concurrent([^[:alnum:]_]|$)|(^|[^[:alnum:]_])race([^[:alnum:]_]|$)|(^|[^[:alnum:]_])atomic([^[:alnum:]_]|$)|(^|[^[:alnum:]_])idempotent([^[:alnum:]_]|$)|(^|[^[:alnum:]_])idempotency([^[:alnum:]_]|$)|(^|[^[:alnum:]_])unique([^[:alnum:]_]|$)|(^|[^[:alnum:]_])constraint([^[:alnum:]_]|$)|(^|[^[:alnum:]_])membership([^[:alnum:]_]|$)|(^|[^[:alnum:]_])member([^[:alnum:]_]|$)|(^|[^[:alnum:]_])organization([^[:alnum:]_]|$)|(^|[^[:alnum:]_])upsert([^[:alnum:]_]|$)'; then add_signal 'database/concurrency'; add_skill 'postgres'; add_skill 'postgresql'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])webhook([^[:alnum:]_]|$)|(^|[^[:alnum:]_])callback([^[:alnum:]_]|$)|(^|[^[:alnum:]_])event([^[:alnum:]_]|$)|(^|[^[:alnum:]_])retry([^[:alnum:]_]|$)|(^|[^[:alnum:]_])signature([^[:alnum:]_]|$)|(^|[^[:alnum:]_])payment([^[:alnum:]_]|$)|(^|[^[:alnum:]_])checkout([^[:alnum:]_]|$)|(^|[^[:alnum:]_])subscription([^[:alnum:]_]|$)|(^|[^[:alnum:]_])order\.created([^[:alnum:]_]|$)|(^|[^[:alnum:]_])order\.paid([^[:alnum:]_]|$)|(^|[^[:alnum:]_])duplicate([^[:alnum:]_]|$)|(^|[^[:alnum:]_])idempotent([^[:alnum:]_]|$)|(^|[^[:alnum:]_])idempotency([^[:alnum:]_]|$)'; then add_signal 'webhook/side-effect flow'; add_skill 'postgres'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])email([^[:alnum:]_]|$)|(^|[^[:alnum:]_])send([^[:alnum:]_]|$)|(^|[^[:alnum:]_])resend([^[:alnum:]_]|$)|(^|[^[:alnum:]_])polar([^[:alnum:]_]|$)|(^|[^[:alnum:]_])analytics([^[:alnum:]_]|$)|(^|[^[:alnum:]_])posthog([^[:alnum:]_]|$)|(^|[^[:alnum:]_])external[[:space:]]+api([^[:alnum:]_]|$)|(^|[^[:alnum:]_])side[[:space:]]+effect([^[:alnum:]_]|$)'; then add_signal 'email/external side effect'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])api([^[:alnum:]_]|$)|(^|[^[:alnum:]_])endpoint([^[:alnum:]_]|$)|(^|[^[:alnum:]_])route([^[:alnum:]_]|$)|(^|[^[:alnum:]_])handler([^[:alnum:]_]|$)|(^|[^[:alnum:]_])procedure([^[:alnum:]_]|$)|(^|[^[:alnum:]_])auth([^[:alnum:]_]|$)|(^|[^[:alnum:]_])session([^[:alnum:]_]|$)|(^|[^[:alnum:]_])invite([^[:alnum:]_]|$)|(^|[^[:alnum:]_])checkout([^[:alnum:]_]|$)|(^|[^[:alnum:]_])recovery([^[:alnum:]_]|$)'; then add_signal 'API/auth flow'; fi

if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])ui([^[:alnum:]_]|$)|(^|[^[:alnum:]_])interface([^[:alnum:]_]|$)|(^|[^[:alnum:]_])component([^[:alnum:]_]|$)|(^|[^[:alnum:]_])page([^[:alnum:]_]|$)|(^|[^[:alnum:]_])screen([^[:alnum:]_]|$)|(^|[^[:alnum:]_])layout([^[:alnum:]_]|$)|(^|[^[:alnum:]_])style([^[:alnum:]_]|$)|(^|[^[:alnum:]_])styling([^[:alnum:]_]|$)|(^|[^[:alnum:]_])design([^[:alnum:]_]|$)|(^|[^[:alnum:]_])visual([^[:alnum:]_]|$)|(^|[^[:alnum:]_])dashboard([^[:alnum:]_]|$)|(^|[^[:alnum:]_])landing([^[:alnum:]_]|$)|(^|[^[:alnum:]_])modal([^[:alnum:]_]|$)|(^|[^[:alnum:]_])dialog([^[:alnum:]_]|$)|(^|[^[:alnum:]_])sidebar([^[:alnum:]_]|$)|(^|[^[:alnum:]_])hero([^[:alnum:]_]|$)|(^|[^[:alnum:]_])card([^[:alnum:]_]|$)|(^|[^[:alnum:]_])table([^[:alnum:]_]|$)'; then add_experience_signal 'UI/interface change: Inspect nearby components/screens before inventing a new visual language. Match the target surface: dashboard/tooling should be dense and scannable; marketing can be more memorable; CLI should stay stable and readable. Avoid generic AI-looking UI unless the project already uses that style intentionally.'; add_skill 'frontend-design'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])form([^[:alnum:]_]|$)|(^|[^[:alnum:]_])field([^[:alnum:]_]|$)|(^|[^[:alnum:]_])input([^[:alnum:]_]|$)|(^|[^[:alnum:]_])submit([^[:alnum:]_]|$)|(^|[^[:alnum:]_])validation([^[:alnum:]_]|$)|(^|[^[:alnum:]_])settings([^[:alnum:]_]|$)|(^|[^[:alnum:]_])profile([^[:alnum:]_]|$)|(^|[^[:alnum:]_])account([^[:alnum:]_]|$)|(^|[^[:alnum:]_])change[[:space:]]+email([^[:alnum:]_]|$)|(^|[^[:alnum:]_])change[[:space:]]+password([^[:alnum:]_]|$)|(^|[^[:alnum:]_])login([^[:alnum:]_]|$)|(^|[^[:alnum:]_])sign[[:space:]]+in([^[:alnum:]_]|$)|(^|[^[:alnum:]_])signup([^[:alnum:]_]|$)|(^|[^[:alnum:]_])checkout([^[:alnum:]_]|$)|(^|[^[:alnum:]_])billing([^[:alnum:]_]|$)|(^|[^[:alnum:]_])wizard([^[:alnum:]_]|$)|(^|[^[:alnum:]_])flow([^[:alnum:]_]|$)'; then add_experience_signal 'form/flow UX: Map the user intent, preconditions, success state, failure state, and recovery path before coding. Cover loading, disabled, pending, success, error, and retry states in the UI. Keep validation server-backed; client validation is for guidance and speed, not trust.'; add_skill 'ux'; add_skill 'frontend-design'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])empty([^[:alnum:]_]|$)|(^|[^[:alnum:]_])loading([^[:alnum:]_]|$)|(^|[^[:alnum:]_])skeleton([^[:alnum:]_]|$)|(^|[^[:alnum:]_])error([^[:alnum:]_]|$)|(^|[^[:alnum:]_])failed([^[:alnum:]_]|$)|(^|[^[:alnum:]_])retry([^[:alnum:]_]|$)|(^|[^[:alnum:]_])disabled([^[:alnum:]_]|$)|(^|[^[:alnum:]_])pending([^[:alnum:]_]|$)|(^|[^[:alnum:]_])expired([^[:alnum:]_]|$)|(^|[^[:alnum:]_])success([^[:alnum:]_]|$)'; then add_experience_signal 'state coverage: Define the state matrix before editing component structure. Make failure and recovery understandable without leaking sensitive backend details. Ensure dynamic state text fits its container across supported viewports.'; add_skill 'ux'; add_skill 'frontend-design'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])accessibility([^[:alnum:]_]|$)|(^|[^[:alnum:]_])a11y([^[:alnum:]_]|$)|(^|[^[:alnum:]_])keyboard([^[:alnum:]_]|$)|(^|[^[:alnum:]_])focus([^[:alnum:]_]|$)|(^|[^[:alnum:]_])aria([^[:alnum:]_]|$)|(^|[^[:alnum:]_])screen[[:space:]]+reader([^[:alnum:]_]|$)|(^|[^[:alnum:]_])contrast([^[:alnum:]_]|$)|(^|[^[:alnum:]_])label([^[:alnum:]_]|$)|(^|[^[:alnum:]_])tab[[:space:]]+order([^[:alnum:]_]|$)'; then add_experience_signal 'accessibility/interaction quality: Use native semantics or accessible primitives before custom controls. Verify labels, focus states, keyboard paths, contrast, and disabled behavior. Do not hide required guidance behind hover-only UI.'; add_skill 'ux'; add_skill 'frontend-design'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])responsive([^[:alnum:]_]|$)|(^|[^[:alnum:]_])mobile([^[:alnum:]_]|$)|(^|[^[:alnum:]_])desktop([^[:alnum:]_]|$)|(^|[^[:alnum:]_])tablet([^[:alnum:]_]|$)|(^|[^[:alnum:]_])breakpoint([^[:alnum:]_]|$)|(^|[^[:alnum:]_])overflow([^[:alnum:]_]|$)|(^|[^[:alnum:]_])truncate([^[:alnum:]_]|$)|(^|[^[:alnum:]_])data[[:space:]]+dense([^[:alnum:]_]|$)|(^|[^[:alnum:]_])table([^[:alnum:]_]|$)'; then add_experience_signal 'responsive layout: Set stable layout constraints before adding dynamic content. Check mobile, desktop, long text, and dense data cases. Prefer predictable navigation and scan paths for operational/product surfaces.'; add_skill 'ux'; add_skill 'frontend-design'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])conversion([^[:alnum:]_]|$)|(^|[^[:alnum:]_])onboarding([^[:alnum:]_]|$)|(^|[^[:alnum:]_])activation([^[:alnum:]_]|$)|(^|[^[:alnum:]_])upgrade([^[:alnum:]_]|$)|(^|[^[:alnum:]_])paywall([^[:alnum:]_]|$)|(^|[^[:alnum:]_])pricing([^[:alnum:]_]|$)|(^|[^[:alnum:]_])cta([^[:alnum:]_]|$)|(^|[^[:alnum:]_])funnel([^[:alnum:]_]|$)|(^|[^[:alnum:]_])adoption([^[:alnum:]_]|$)'; then add_experience_signal 'conversion/onboarding flow: Clarify the next action and the reason to continue. Keep measurement and UI state aligned with the existing product analytics pattern. Do not replace in-app UX with broad marketing copy unless the task is explicitly a marketing surface.'; add_skill 'ux'; add_skill 'frontend-design'; fi

if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])cloudflare([^[:alnum:]_]|$)|(^|[^[:alnum:]_])worker([^[:alnum:]_]|$)|(^|[^[:alnum:]_])wrangler([^[:alnum:]_]|$)|(^|[^[:alnum:]_])alchemy\.run\.ts([^[:alnum:]_]|$)|(^|[^[:alnum:]_])@cloudflare/workers-types([^[:alnum:]_]|$)'; then add_provider_docs 'Cloudflare Workers: Search Cloudflare docs for the task capability before choosing a fallback. Service catalog: https://developers.cloudflare.com/products/ Runtime docs: https://developers.cloudflare.com/workers/ Best practices: https://developers.cloudflare.com/workers/platform/best-practices/'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])vercel([^[:alnum:]_]|$)|(^|[^[:alnum:]_])vercel\.json([^[:alnum:]_]|$)|(^|[^[:alnum:]_])@vercel/([^[:alnum:]_]|$)'; then add_provider_docs 'Vercel: Search Vercel docs for the task capability and runtime recommendation before choosing a fallback. Service catalog: https://vercel.com/docs Runtime docs: https://vercel.com/docs/functions Best practices: https://vercel.com/docs/frameworks'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])polar([^[:alnum:]_]|$)|(^|[^[:alnum:]_])@polar-sh([^[:alnum:]_]|$)|(^|[^[:alnum:]_])polar_([^[:alnum:]_]|$)'; then add_provider_docs 'Polar: Search Polar docs and the SDK webhook helpers before handling payment, subscription, or checkout events. Service catalog: https://polar.sh/docs Runtime docs: https://polar.sh/docs/integrate/webhooks/endpoints Best practices: https://polar.sh/docs/integrate/webhooks/delivery'; fi
if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])alchemy([^[:alnum:]_]|$)|(^|[^[:alnum:]_])alchemy\.run\.ts([^[:alnum:]_]|$)'; then add_provider_docs 'Alchemy IaC: Search Alchemy provider/resource docs for how this repo declares runtime services and bindings. Service catalog: https://alchemy.run/ Runtime docs: https://alchemy.run/'; fi

if [ -n "$RISK_SIGNALS" ]; then
  if [ -f "packages/infra/alchemy.run.ts" ] || [ -f "alchemy.run.ts" ] || find . -maxdepth 4 \( -name 'wrangler.toml' -o -name 'wrangler.json' -o -name 'wrangler.jsonc' \) -print -quit 2>/dev/null | grep -q . || grep -R "Cloudflare Workers\|@cloudflare/workers-types" package.json packages apps 2>/dev/null | head -n 1 | grep -q .; then
    add_skill 'cloudflare-workers'
    add_provider_docs 'Cloudflare Workers: Search Cloudflare docs for the task capability before choosing a fallback. Service catalog: https://developers.cloudflare.com/products/ Runtime docs: https://developers.cloudflare.com/workers/ Best practices: https://developers.cloudflare.com/workers/platform/best-practices/ Rate limiting docs: https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/'
  fi
  if [ -f "packages/infra/alchemy.run.ts" ] || [ -f "alchemy.run.ts" ] || grep -R '"alchemy"' package.json packages apps 2>/dev/null | head -n 1 | grep -q .; then
    add_skill 'alchemy'
    add_provider_docs 'Alchemy IaC: Search Alchemy provider/resource docs for how this repo declares runtime services and bindings. Service catalog: https://alchemy.run/ Runtime docs: https://alchemy.run/'
  fi
fi

if printf '%s' "$TEXT" | grep -Eiq '(^|[^[:alnum:]_])do[[:space:]]+not[[:space:]]+use([^[:alnum:]_]|$)|(^|[^[:alnum:]_])don[[:space:]]+t[[:space:]]+use([^[:alnum:]_]|$)|(^|[^[:alnum:]_])without([^[:alnum:]_]|$)|(^|[^[:alnum:]_])avoid([^[:alnum:]_]|$)|(^|[^[:alnum:]_])must[[:space:]]+use([^[:alnum:]_]|$)|(^|[^[:alnum:]_])use[[:space:]]+postgres([^[:alnum:]_]|$)|(^|[^[:alnum:]_])use[[:space:]]+redis([^[:alnum:]_]|$)|(^|[^[:alnum:]_])use[[:space:]]+database([^[:alnum:]_]|$)|(^|[^[:alnum:]_])do[[:space:]]+not[[:space:]]+touch[[:space:]]+infra([^[:alnum:]_]|$)|(^|[^[:alnum:]_])no[[:space:]]+cloudflare([^[:alnum:]_]|$)|(^|[^[:alnum:]_])no[[:space:]]+bindings([^[:alnum:]_]|$)'; then EXPLICIT_OVERRIDE="1"; fi

if printf '%s' "$RISK_SIGNALS" | grep -Eiq 'abuse/rate-limit guard' && printf '%s' "$PROVIDER_DOCS" | grep -Eiq 'Cloudflare Workers'; then
  add_preferred_service 'Cloudflare Workers Rate limiting: prefer RateLimit binding through infrastructure/runtime config; docs https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/; avoid Map/request-local counters and DB fallback unless explicitly overridden.'
fi

if [ -z "$LIKELY_SKILLS" ] && [ -z "$RISK_SIGNALS" ] && [ -z "$EXPERIENCE_SIGNALS" ] && [ -z "$PROVIDER_DOCS" ]; then
  exit 0
fi

if [ -n "$EXPLICIT_OVERRIDE" ]; then
  OVERRIDE_NOTE="
Explicit user override detected: follow the user preference. Still mention the provider-recommended path if relevant, and make the chosen fallback safe for concurrency, side effects, and failure modes."
fi

if [ -n "$RISK_SIGNALS$EXPERIENCE_SIGNALS" ]; then
  mkdir -p "$HOOK_DIR" 2>/dev/null
  {
    printf 'risk:%s\n' "$RISK_SIGNALS"
    printf 'experience:%s\n' "$EXPERIENCE_SIGNALS"
    printf 'override:%s\n' "$EXPLICIT_OVERRIDE"
    printf 'skills:%s\n' "$LIKELY_SKILLS"
    printf 'providers:%s\n' "$PROVIDER_DOCS"
    printf 'preferred:%s\n' "$PREFERRED_SERVICES"
  } > "$CONTRACT_PATH" 2>/dev/null
fi

cat <<EOF
SUMMONAIKIT SKILL PREFLIGHT

Likely skills to inspect/load:$LIKELY_SKILLS

Production risk scan:$RISK_SIGNALS

UI/UX quality scan:$EXPERIENCE_SIGNALS

Provider docs/service discovery:$PROVIDER_DOCS
$OVERRIDE_NOTE

CAPABILITY TASK CONTRACT

Likely skills:$LIKELY_SKILLS
Preferred native services:$PREFERRED_SERVICES

UI/UX QUALITY CONTRACT

Required checks:
- Inspect nearby screens/components before inventing new structure, styling, or copy.
- Identify the state matrix for changed interactive surfaces: loading, empty, error, disabled, pending, success, and recovery.
- Preserve the project's component library, design tokens, motion style, density, and interaction patterns.
- Avoid generic AI slop, template-looking layouts, ungrounded gradient/card stacks, and one-size-fits-all UI.
- Verify responsive behavior, accessibility basics, long text, focus/keyboard paths, and failure copy before finishing.

Required checks:
- Inspect provider service catalog/docs and project runtime config before selecting a fallback.
- Wire platform capabilities through infrastructure/runtime config, typed env/context, and the handler/procedure boundary.
- Guard before external payment APIs, auth mutations, email sends, analytics events, storage writes, and database mutations.
- Fallbacks must be durable, multi-instance safe, and atomic under concurrency.
- The final hook check will flag missing native primitives, non-atomic fallback counters, and serverless anti-patterns.

Before coding:
- Load or explicitly inspect the likely skills that apply to this task.
- If any production risk appears, inspect the provider service catalog, best-practice docs, runtime/database config, and project docs before choosing an implementation.
- If any UI/UX signal appears, inspect nearby screens/components, identify states, preserve design system conventions, and avoid generic AI-looking UI.
- If the user or project docs clearly require a different mechanism, follow that requirement and call out the tradeoff.
- Otherwise prefer the platform-recommended or durable primitive before in-memory, frontend-only, detached async, or ad hoc counter solutions.
- Place guards before external side effects and document failure behavior.
EOF
