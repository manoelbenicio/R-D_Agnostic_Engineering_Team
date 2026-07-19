//! rpp.l2.v1 sidecar binary for the Multica Go daemon.
//!
//! This broker exposes the control surface the Multica Go daemon and shell
//! smokes expect, and bridges runtime traffic to a real `prodex gateway`
//! subprocess so Smart Context and provider routing are owned by prodex.

use chrono::Utc;
use serde::Serialize;
use serde_json::json;
use std::collections::HashMap;
use std::io::{Read, Write};
use std::net::{SocketAddr, TcpStream};
use std::path::Path;
use std::process::{Child, Command, Stdio};
use std::sync::{Arc, Mutex, OnceLock};
use std::time::Duration;
use tiny_http::{Header, Method, Request, Response, ResponseBox, Server, StatusCode};

const DEFAULT_BIND: &str = "127.0.0.1:43117";
const CONTRACT_VERSION: &str = "rpp.l2.v1";

#[derive(Clone, Debug)]
struct KillSwitchRecord {
    tenant_id: String,
    provider: String,
    profile_id: String,
    session_id: String,
    feature: String,
    state: String,
    effective_at: String,
}

#[derive(Default)]
struct SidecarState {
    policies: HashMap<String, serde_json::Value>,
    registered: HashMap<String, serde_json::Value>,
    sessions: HashMap<String, RuntimeSession>,
    kill_switches: Vec<KillSwitchRecord>,
}

#[derive(Clone, Debug)]
struct RuntimeSession {
    runtime_session_id: String,
    tenant_id: String,
    session_id: String,
    provider: String,
    profile_id: String,
    gateway_addr: String,
}

struct GatewayProcess {
    child: Child,
    addr: String,
    token: String,
}

fn state() -> &'static Arc<Mutex<SidecarState>> {
    static STATE: OnceLock<Arc<Mutex<SidecarState>>> = OnceLock::new();
    STATE.get_or_init(|| Arc::new(Mutex::new(SidecarState::default())))
}

fn event_queues() -> &'static Arc<Mutex<HashMap<String, Vec<serde_json::Value>>>> {
    static QUEUES: OnceLock<Arc<Mutex<HashMap<String, Vec<serde_json::Value>>>>> = OnceLock::new();
    QUEUES.get_or_init(|| Arc::new(Mutex::new(HashMap::new())))
}

fn gateway_process() -> &'static Arc<Mutex<Option<GatewayProcess>>> {
    static GATEWAY: OnceLock<Arc<Mutex<Option<GatewayProcess>>>> = OnceLock::new();
    GATEWAY.get_or_init(|| Arc::new(Mutex::new(None)))
}

fn sidecar_bind_addr() -> &'static Arc<Mutex<String>> {
    static BIND: OnceLock<Arc<Mutex<String>>> = OnceLock::new();
    BIND.get_or_init(|| Arc::new(Mutex::new(DEFAULT_BIND.to_string())))
}

fn required_token() -> &'static String {
    static TOKEN: OnceLock<String> = OnceLock::new();
    TOKEN.get_or_init(|| std::env::var("MULTICA_L2_BEARER_TOKEN").unwrap_or_default())
}

fn allowed_tenants() -> &'static Option<Vec<String>> {
    static TENANTS: OnceLock<Option<Vec<String>>> = OnceLock::new();
    TENANTS.get_or_init(|| {
        let raw = std::env::var("MULTICA_L2_ALLOWED_TENANTS").unwrap_or_default();
        let tenants: Vec<String> = raw
            .split(',')
            .map(str::trim)
            .filter(|v| !v.is_empty())
            .map(ToOwned::to_owned)
            .collect();
        if tenants.is_empty() {
            None
        } else {
            Some(tenants)
        }
    })
}

fn generate_id(prefix: &str) -> String {
    let ts = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos();
    format!("{prefix}-{ts}")
}

fn content_type(value: &[u8]) -> Header {
    Header::from_bytes(&b"Content-Type"[..], value).unwrap()
}

fn json_response<T: Serialize>(value: &T) -> ResponseBox {
    json_response_status(200, value)
}

fn json_response_status<T: Serialize>(status: u16, value: &T) -> ResponseBox {
    Response::from_string(serde_json::to_string(value).unwrap())
        .with_status_code(StatusCode(status))
        .with_header(content_type(b"application/json"))
        .boxed()
}

fn error_response(status: u16, msg: &str) -> ResponseBox {
    Response::from_string(json!({"error": msg}).to_string())
        .with_status_code(StatusCode(status))
        .with_header(content_type(b"application/json"))
        .boxed()
}

fn check_auth(req: &Request) -> bool {
    let token = required_token();
    if token.is_empty() {
        return false;
    }
    let expected = format!("Bearer {token}");
    req.headers().iter().any(|h| {
        h.field
            .as_str()
            .as_str()
            .eq_ignore_ascii_case("Authorization")
            && h.value.as_str() == expected
    })
}

fn read_json_body(req: &mut Request) -> Result<serde_json::Value, String> {
    let mut body = String::new();
    req.as_reader()
        .read_to_string(&mut body)
        .map_err(|e| e.to_string())?;
    serde_json::from_str(&body).map_err(|e| e.to_string())
}

fn require_contract_version(value: &serde_json::Value) -> Result<(), ResponseBox> {
    if value.get("contract_version").and_then(|v| v.as_str()) != Some(CONTRACT_VERSION) {
        return Err(error_response(400, "contract_version must be rpp.l2.v1"));
    }
    Ok(())
}

fn require_allowed_tenant(value: &serde_json::Value) -> Result<(), ResponseBox> {
    let tenant_id = value
        .get("tenant_id")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if tenant_id.is_empty() {
        return Err(error_response(400, "tenant_id is required"));
    }
    if let Some(allowed) = allowed_tenants() {
        if !allowed.iter().any(|tenant| tenant == tenant_id) {
            return Err(error_response(403, "tenant is not registered"));
        }
    }
    Ok(())
}

fn handle_healthz() -> ResponseBox {
    json_response(&json!({
        "contract_version": CONTRACT_VERSION,
        "status": "alive",
        "sidecar": {
            "name": "prodex-sidecar",
            "version": env!("CARGO_PKG_VERSION"),
            "commit": std::env::var("MULTICA_PRODEX_COMMIT").unwrap_or_else(|_| "smoke".to_string())
        }
    }))
}

fn handle_readyz() -> ResponseBox {
    let pg = probe_postgres();
    let gateway = ensure_gateway_running()
        .and_then(|gw| probe_gateway(&gw.addr).map(|_| gw))
        .map(|gw| ProbeResult::pass(json!({"pid": gateway_pid(), "listen_addr": gw.addr})))
        .unwrap_or_else(|err| ProbeResult::fail(err));
    let ready = pg.ok && gateway.ok;
    let body = json!({
        "contract_version": CONTRACT_VERSION,
        "status": if ready { "ready" } else { "error" },
        "checks": [
            {
                "name": "shared_state_backend",
                "status": if pg.ok { "pass" } else { "fail" },
                "details": pg.details
            },
            {"name": "kill_switch", "status": "pass"},
            {
                "name": "runtime_proxy",
                "status": if gateway.ok { "pass" } else { "fail" },
                "details": gateway.details
            },
            {"name": "event_stream", "status": "pass"}
        ]
    });
    emit_event(
        "",
        "health_status",
        json!({
            "producer_component": "sidecar",
            "severity": if ready { "info" } else { "error" },
            "message": if ready { "readyz pass" } else { "readyz fail" }
        }),
    );
    json_response_status(if ready { 200 } else { 503 }, &body)
}

struct ProbeResult {
    ok: bool,
    details: serde_json::Value,
}

impl ProbeResult {
    fn pass(details: serde_json::Value) -> Self {
        Self { ok: true, details }
    }

    fn fail(error: String) -> Self {
        Self {
            ok: false,
            details: json!({
                "connection_status": "error",
                "error": scrub_error_category(&error)
            }),
        }
    }
}

fn probe_postgres() -> ProbeResult {
    let Ok(url) = std::env::var("PRODEX_PG_URL") else {
        return ProbeResult::fail("missing_config".to_string());
    };
    if url.trim().is_empty() {
        return ProbeResult::fail("missing_config".to_string());
    }
    match Command::new("psql")
        .arg(url)
        .args(["-Atqc", "SELECT 1"])
        .env("PGCONNECT_TIMEOUT", "2")
        .stdin(Stdio::null())
        .output()
    {
        Ok(output)
            if output.status.success() && String::from_utf8_lossy(&output.stdout).trim() == "1" =>
        {
            ProbeResult::pass(json!({
                "backend_type": "postgres",
                "configured": true,
                "probe": "SELECT 1",
                "connection_status": "ok"
            }))
        }
        Ok(output) if output.status.success() => ProbeResult::fail("query_failed".to_string()),
        Ok(_) => ProbeResult::fail("connect_failed".to_string()),
        Err(_) => ProbeResult::fail("psql_unavailable".to_string()),
    }
}

fn scrub_error_category(error: &str) -> &'static str {
    match error {
        "missing_config" => "missing_config",
        "query_failed" => "query_failed",
        "psql_unavailable" => "psql_unavailable",
        "gateway_not_started" => "gateway_not_started",
        "gateway_exited" => "gateway_exited",
        "gateway_port_closed" => "gateway_port_closed",
        "spawn_failed" => "spawn_failed",
        _ => "connect_failed",
    }
}

fn handle_policy_apply(value: &serde_json::Value) -> ResponseBox {
    let policy_id = value
        .get("policy_id")
        .and_then(|v| v.as_str())
        .unwrap_or("default")
        .to_string();
    let revision = value.get("revision").and_then(|v| v.as_i64()).unwrap_or(1);
    state()
        .lock()
        .unwrap()
        .policies
        .insert(policy_id.clone(), value.clone());
    let session_id = value
        .get("session_id")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    emit_event(
        session_id,
        "policy_applied",
        json!({
            "producer_component": "policy",
            "tenant_id": value.get("tenant_id").and_then(|v| v.as_str()).unwrap_or("default"),
            "policy_id": policy_id,
            "message": format!("policy applied revision={revision}")
        }),
    );
    json_response(&json!({
        "contract_version": CONTRACT_VERSION,
        "request_id": value.get("request_id").and_then(|v| v.as_str()).unwrap_or(""),
        "policy_id": policy_id,
        "revision": revision,
        "applied": true
    }))
}

fn handle_accounts_register(value: &serde_json::Value) -> ResponseBox {
    let profiles = value
        .get("profiles")
        .and_then(|p| p.as_array())
        .cloned()
        .unwrap_or_default();
    let mut rejected = Vec::new();
    let mut registered_count = 0usize;

    for profile in profiles {
        let id = profile
            .get("profile_id")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string();
        let home = profile
            .get("profile_home")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        if id.is_empty()
            || !is_managed_profile_home(home)
            || contains_forbidden_secret_field(&profile)
        {
            if !id.is_empty() {
                rejected.push(id);
            }
            continue;
        }
        registered_count += 1;
        state().lock().unwrap().registered.insert(id, profile);
    }

    json_response(&json!({
        "contract_version": CONTRACT_VERSION,
        "request_id": value.get("request_id").and_then(|v| v.as_str()).unwrap_or(""),
        "registered_profile_count": registered_count,
        "rejected_profiles": rejected
    }))
}

fn is_managed_profile_home(home: &str) -> bool {
    let managed_roots = ["/tmp/rpp-smoke", "/home", "/root/rpp"];
    managed_roots
        .iter()
        .any(|root| home.starts_with(root) && !home.contains("outside-managed-root"))
        && Path::new(home).is_dir()
}

fn contains_forbidden_secret_field(value: &serde_json::Value) -> bool {
    match value {
        serde_json::Value::Object(map) => map.iter().any(|(key, nested)| {
            matches!(
                key.as_str(),
                "api_key"
                    | "access_token"
                    | "refresh_token"
                    | "bearer_token"
                    | "cookie"
                    | "cookies"
                    | "auth_json"
                    | "auth"
                    | "raw_auth"
            ) || contains_forbidden_secret_field(nested)
        }),
        serde_json::Value::Array(items) => items.iter().any(contains_forbidden_secret_field),
        _ => false,
    }
}

fn handle_killswitch_apply(value: &serde_json::Value) -> ResponseBox {
    let record = KillSwitchRecord {
        tenant_id: value
            .get("tenant_id")
            .and_then(|v| v.as_str())
            .unwrap_or("default")
            .to_string(),
        provider: value
            .pointer("/scope/provider")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        profile_id: value
            .pointer("/scope/profile_id")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        session_id: value
            .pointer("/scope/session_id")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        feature: value
            .get("feature")
            .and_then(|v| v.as_str())
            .unwrap_or("smart_context")
            .to_string(),
        state: value
            .get("state")
            .and_then(|v| v.as_str())
            .unwrap_or("disabled")
            .to_string(),
        effective_at: value
            .get("effective_at")
            .and_then(|v| v.as_str())
            .unwrap_or("next_request")
            .to_string(),
    };

    {
        let mut st = state().lock().unwrap();
        st.kill_switches
            .retain(|existing| !same_switch(existing, &record));
        if record.state == "disabled" {
            st.kill_switches.push(record.clone());
        }
    }

    emit_event(
        &record.session_id,
        "killswitch_toggled",
        json!({
            "tenant_id": record.tenant_id,
            "kill_switch": {
                "scope": kill_switch_scope(&record),
                "feature": record.feature,
                "enabled": record.state != "disabled"
            },
            "feature": record.feature,
            "state": record.state,
            "message": format!("kill switch {} {}", record.feature, record.state)
        }),
    );

    json_response(&json!({
        "contract_version": CONTRACT_VERSION,
        "request_id": value.get("request_id").and_then(|v| v.as_str()).unwrap_or(""),
        "applied": true,
        "effective_at": record.effective_at
    }))
}

fn kill_switch_scope(record: &KillSwitchRecord) -> &'static str {
    if !record.session_id.is_empty() {
        "session"
    } else if !record.profile_id.is_empty() {
        "profile"
    } else if !record.provider.is_empty() {
        "provider"
    } else if !record.tenant_id.is_empty() {
        "tenant"
    } else {
        "global"
    }
}

fn same_switch(a: &KillSwitchRecord, b: &KillSwitchRecord) -> bool {
    a.tenant_id == b.tenant_id
        && a.provider == b.provider
        && a.profile_id == b.profile_id
        && a.session_id == b.session_id
        && a.feature == b.feature
}

fn handle_killswitch_status(path: &str) -> ResponseBox {
    let query = path.split_once('?').map(|(_, q)| q).unwrap_or("");
    let params = parse_query(query);
    let tenant_id = params
        .get("tenant_id")
        .map(String::as_str)
        .unwrap_or("default");
    let provider = params.get("provider").map(String::as_str).unwrap_or("");
    let profile_id = params.get("profile_id").map(String::as_str).unwrap_or("");
    let session_id = params.get("session_id").map(String::as_str).unwrap_or("");
    let feature = params.get("feature").map(String::as_str).unwrap_or("");
    let active = switch_applies(tenant_id, provider, profile_id, session_id, feature);
    json_response(&json!({
        "contract_version": CONTRACT_VERSION,
        "active": active,
        "tenant_id": tenant_id,
        "provider": provider,
        "profile_id": profile_id,
        "session_id": session_id,
        "feature": feature
    }))
}

fn parse_query(query: &str) -> HashMap<String, String> {
    query
        .split('&')
        .filter_map(|part| part.split_once('='))
        .map(|(k, v)| (k.to_string(), v.to_string()))
        .collect()
}

fn switch_applies(
    tenant_id: &str,
    provider: &str,
    profile_id: &str,
    session_id: &str,
    feature: &str,
) -> bool {
    state().lock().unwrap().kill_switches.iter().any(|record| {
        record.tenant_id == tenant_id
            && (feature.is_empty() || record.feature == feature)
            && (record.provider.is_empty() || record.provider == provider)
            && (record.profile_id.is_empty() || record.profile_id == profile_id)
            && (record.session_id.is_empty() || record.session_id == session_id)
    })
}

fn session_block(value: &serde_json::Value) -> Option<String> {
    let tenant_id = value
        .get("tenant_id")
        .and_then(|v| v.as_str())
        .unwrap_or("default");
    let provider = value
        .get("requested_provider")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let session_id = value
        .get("session_id")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let profile_id = value
        .get("profile_pool")
        .and_then(|v| v.as_array())
        .and_then(|profiles| profiles.first())
        .and_then(|v| v.as_str())
        .unwrap_or("");
    for feature in ["runtime_proxy", "gateway", "provider_bridge"] {
        if switch_applies(tenant_id, provider, profile_id, session_id, feature) {
            return Some(format!("{feature} disabled by kill switch"));
        }
    }
    None
}

fn handle_session_start(value: &serde_json::Value) -> ResponseBox {
    if let Some(reason) = session_block(value) {
        return error_response(423, &reason);
    }

    let session_id = value
        .get("session_id")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown")
        .to_string();
    let tenant_id = value
        .get("tenant_id")
        .and_then(|v| v.as_str())
        .unwrap_or("default");
    let provider = value
        .get("requested_provider")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let profile_id = value
        .get("profile_pool")
        .and_then(|v| v.as_array())
        .and_then(|profiles| profiles.first())
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let smart_context_mode = if switch_applies(
        tenant_id,
        provider,
        profile_id,
        &session_id,
        "smart_context",
    ) {
        "exact"
    } else {
        "proxy_rewrite"
    };
    let gateway = match ensure_gateway_running() {
        Ok(gateway) => gateway,
        Err(err) => {
            return error_response(
                503,
                &format!(
                    "runtime gateway unavailable: {}",
                    scrub_error_category(&err)
                ),
            )
        }
    };
    let runtime_session_id = generate_id("rt");
    let base_url = sidecar_base_url();
    let event_stream_url = format!("{base_url}/v1/events/stream?session_id={session_id}");
    let runtime_endpoint = format!("{base_url}/v1/runtime/proxy?session_id={session_id}");

    state().lock().unwrap().sessions.insert(
        session_id.clone(),
        RuntimeSession {
            runtime_session_id: runtime_session_id.clone(),
            tenant_id: tenant_id.to_string(),
            session_id: session_id.clone(),
            provider: if provider.is_empty() {
                "openai-compatible"
            } else {
                provider
            }
            .to_string(),
            profile_id: profile_id.to_string(),
            gateway_addr: gateway.addr.clone(),
        },
    );
    event_queues()
        .lock()
        .unwrap()
        .entry(session_id.clone())
        .or_default();
    emit_event(
        &session_id,
        "session_started",
        json!({
            "tenant_id": tenant_id,
            "runtime_session_id": runtime_session_id,
            "provider": if provider.is_empty() { "openai-compatible" } else { provider },
            "profile_id": if profile_id.is_empty() { "profile-default" } else { profile_id },
            "message": "session started with prodex gateway smart context"
        }),
    );

    json_response(&json!({
        "contract_version": CONTRACT_VERSION,
        "request_id": value.get("request_id").and_then(|v| v.as_str()).unwrap_or(""),
        "runtime_session_id": runtime_session_id,
        "router_owner": "rust_l2",
        "event_stream_url": event_stream_url,
        "runtime_endpoint": runtime_endpoint,
        "runtime_log_ref": format!("prodex-gateway://{}", gateway.addr),
        "smart_context_mode": smart_context_mode,
        "gateway": {
            "listen_addr": gateway.addr,
            "smart_context_enabled": true
        }
    }))
}

fn handle_session_stop(value: &serde_json::Value) -> ResponseBox {
    let session_id = value
        .get("session_id")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown")
        .to_string();
    state().lock().unwrap().sessions.remove(&session_id);
    emit_event(
        &session_id,
        "session_stopped",
        json!({
            "tenant_id": value.get("tenant_id").and_then(|v| v.as_str()).unwrap_or("default"),
            "message": "session stopped"
        }),
    );
    json_response(&json!({
        "contract_version": CONTRACT_VERSION,
        "request_id": value.get("request_id").and_then(|v| v.as_str()).unwrap_or(""),
        "stopped": true
    }))
}

fn emit_event(session_id: &str, event_type: &str, payload: serde_json::Value) {
    let mut event = json!({
        "contract_version": CONTRACT_VERSION,
        "event_id": generate_id("evt"),
        "event_type": event_type,
        "occurred_at": Utc::now().to_rfc3339(),
        "severity": payload.get("severity").and_then(|v| v.as_str()).unwrap_or("info"),
        "producer": {
            "plane": "rust_l2",
            "component": payload.get("producer_component").and_then(|v| v.as_str()).unwrap_or("event_stream")
        },
        "tenant_id": "default",
        "session_id": session_id,
        "redaction": {"secrets_present": false, "scrubber_version": "sidecar-smoke-1.0.0"}
    });
    if let Some(map) = payload.as_object() {
        let event_map = event.as_object_mut().unwrap();
        for key in [
            "tenant_id",
            "workspace_id",
            "task_id",
            "runtime_session_id",
            "runtime_request_id",
            "policy_id",
            "profile_id",
            "provider",
            "route_decision",
            "tool_call_id",
            "tool_name",
            "continuation",
            "kill_switch",
            "message",
        ] {
            if let Some(value) = map.get(key) {
                event_map.insert(key.to_string(), value.clone());
            }
        }
    }
    if session_id.is_empty() {
        event.as_object_mut().unwrap().remove("session_id");
    }
    event_queues()
        .lock()
        .unwrap()
        .entry(session_id.to_string())
        .or_default()
        .push(event);
}

fn handle_events_stream(path: &str) -> ResponseBox {
    let query = path.split_once('?').map(|(_, q)| q).unwrap_or("");
    let params = parse_query(query);
    let session_id = params
        .get("session_id")
        .cloned()
        .unwrap_or_else(|| "session-smoke".to_string());
    let mut lines = String::new();
    let mut queues = event_queues().lock().unwrap();
    for event in queues.entry(session_id).or_default().drain(..) {
        lines.push_str(&event.to_string());
        lines.push('\n');
    }
    Response::from_data(lines.into_bytes())
        .with_header(content_type(b"application/x-ndjson"))
        .boxed()
}

fn handle_runtime_proxy(req: &mut Request, path: &str) -> ResponseBox {
    let raw_body = match read_body_bytes(req) {
        Ok(body) => body,
        Err(err) => return error_response(400, &err),
    };
    let query = path.split_once('?').map(|(_, q)| q).unwrap_or("");
    let params = parse_query(query);
    let envelope: serde_json::Value =
        serde_json::from_slice(&raw_body).unwrap_or_else(|_| json!({}));
    let session_id = params
        .get("session_id")
        .cloned()
        .or_else(|| {
            envelope
                .get("session_id")
                .and_then(|v| v.as_str())
                .map(ToOwned::to_owned)
        })
        .unwrap_or_else(|| "unknown".to_string());
    let session = match state().lock().unwrap().sessions.get(&session_id).cloned() {
        Some(session) => session,
        None => return error_response(404, "unknown runtime session"),
    };
    let gateway_path = envelope
        .get("gateway_path")
        .and_then(|v| v.as_str())
        .unwrap_or("/v1/responses");
    let request_id = envelope
        .get("request_id")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let runtime_request_id = envelope
        .get("runtime_request_id")
        .and_then(|v| v.as_str())
        .map(ToOwned::to_owned)
        .unwrap_or_else(|| generate_id("rt-req"));
    let gateway_body = envelope
        .get("body")
        .cloned()
        .map(|v| serde_json::to_vec(&v).unwrap_or_default())
        .unwrap_or(raw_body);
    let before_tokens = estimate_tokens(&gateway_body);
    let gateway = match ensure_gateway_running() {
        Ok(gateway) => gateway,
        Err(err) => {
            return error_response(
                503,
                &format!(
                    "runtime gateway unavailable: {}",
                    scrub_error_category(&err)
                ),
            )
        }
    };
    let auth_header = format!("Bearer {}", gateway.token);
    let response = match http_request(
        &gateway.addr,
        "POST",
        gateway_path,
        &[
            ("Authorization", auth_header.as_str()),
            ("Content-Type", "application/json"),
        ],
        &gateway_body,
    ) {
        Ok(resp) => resp,
        Err(err) => {
            return error_response(
                502,
                &format!("gateway proxy failed: {}", scrub_error_category(&err)),
            )
        }
    };
    let gateway_json = serde_json::from_slice::<serde_json::Value>(&response.body).ok();
    let after_tokens = gateway_json
        .as_ref()
        .and_then(extract_usage_input_tokens)
        .unwrap_or_else(|| estimate_tokens(&response.body));
    let reduction_percent = token_reduction_percent(before_tokens, after_tokens);
    let committed = response.status < 500;

    emit_event(
        &session.session_id,
        "route_decision",
        json!({
            "producer_component": "runtime_proxy",
            "tenant_id": session.tenant_id,
            "runtime_session_id": session.runtime_session_id,
            "runtime_request_id": runtime_request_id,
            "provider": session.provider,
            "profile_id": if session.profile_id.is_empty() { "profile-default" } else { &session.profile_id },
            "route_decision": {
                "decision_phase": "pre_commit",
                "selected_profile_id": if session.profile_id.is_empty() { "profile-default" } else { &session.profile_id },
                "selected_provider": session.provider,
                "reason": "prodex_gateway_smart_context",
                "committed": committed
            },
            "message": format!(
                "smart_context tokens before={} after={} reduction_percent={}",
                before_tokens, after_tokens, reduction_percent
            )
        }),
    );

    let body = json!({
        "contract_version": CONTRACT_VERSION,
        "request_id": request_id,
        "session_id": session.session_id,
        "runtime_session_id": session.runtime_session_id,
        "runtime_request_id": runtime_request_id,
        "router_owner": "rust_l2",
        "gateway_status": response.status,
        "smart_context": {
            "mode": "proxy_rewrite",
            "gateway_addr": session.gateway_addr,
            "input_tokens_before_estimate": before_tokens,
            "input_tokens_after_observed_or_estimate": after_tokens,
            "input_token_reduction_percent": reduction_percent,
            "measurement_source": if gateway_json.as_ref().and_then(extract_usage_input_tokens).is_some() { "gateway_usage" } else { "local_estimate" }
        },
        "gateway_response": gateway_json.unwrap_or_else(|| json!({
            "body_bytes": response.body.len(),
            "body_redacted": true
        }))
    });
    json_response_status(if committed { 200 } else { 502 }, &body)
}

fn read_body_bytes(req: &mut Request) -> Result<Vec<u8>, String> {
    let mut body = Vec::new();
    req.as_reader()
        .read_to_end(&mut body)
        .map_err(|e| e.to_string())?;
    Ok(body)
}

struct HttpResponse {
    status: u16,
    body: Vec<u8>,
}

fn http_request(
    addr: &str,
    method: &str,
    path: &str,
    headers: &[(&str, &str)],
    body: &[u8],
) -> Result<HttpResponse, String> {
    let mut stream = connect_loopback(addr)?;
    let mut request = format!(
        "{method} {path} HTTP/1.1\r\nHost: {addr}\r\nConnection: close\r\nContent-Length: {}\r\n",
        body.len()
    );
    for (name, value) in headers {
        request.push_str(name);
        request.push_str(": ");
        request.push_str(value);
        request.push_str("\r\n");
    }
    request.push_str("\r\n");
    stream
        .write_all(request.as_bytes())
        .and_then(|_| stream.write_all(body))
        .map_err(|_| "connect_failed".to_string())?;
    let mut raw = Vec::new();
    stream
        .read_to_end(&mut raw)
        .map_err(|_| "connect_failed".to_string())?;
    parse_http_response(&raw)
}

fn parse_http_response(raw: &[u8]) -> Result<HttpResponse, String> {
    let text = String::from_utf8_lossy(raw);
    let Some((head, body)) = text.split_once("\r\n\r\n") else {
        return Err("connect_failed".to_string());
    };
    let status = head
        .lines()
        .next()
        .and_then(|line| line.split_whitespace().nth(1))
        .and_then(|code| code.parse::<u16>().ok())
        .ok_or_else(|| "connect_failed".to_string())?;
    Ok(HttpResponse {
        status,
        body: body.as_bytes().to_vec(),
    })
}

fn connect_loopback(addr: &str) -> Result<TcpStream, String> {
    if !(addr.starts_with("127.0.0.1:") || addr.starts_with("localhost:")) {
        return Err("connect_failed".to_string());
    }
    let socket: SocketAddr = addr
        .replace("localhost", "127.0.0.1")
        .parse()
        .map_err(|_| "connect_failed".to_string())?;
    let stream = TcpStream::connect_timeout(&socket, Duration::from_secs(2))
        .map_err(|_| "gateway_port_closed".to_string())?;
    let _ = stream.set_read_timeout(Some(Duration::from_secs(5)));
    let _ = stream.set_write_timeout(Some(Duration::from_secs(5)));
    Ok(stream)
}

fn probe_gateway(addr: &str) -> Result<(), String> {
    connect_loopback(addr).map(|_| ())
}

fn gateway_pid() -> u32 {
    gateway_process()
        .lock()
        .unwrap()
        .as_ref()
        .map(|gateway| gateway.child.id())
        .unwrap_or(0)
}

#[derive(Clone)]
struct GatewayHandle {
    addr: String,
    token: String,
}

fn ensure_gateway_running() -> Result<GatewayHandle, String> {
    let mut guard = gateway_process().lock().unwrap();
    if let Some(gateway) = guard.as_mut() {
        match gateway.child.try_wait() {
            Ok(Some(_)) => {
                *guard = None;
                return Err("gateway_exited".to_string());
            }
            Ok(None) => {
                return Ok(GatewayHandle {
                    addr: gateway.addr.clone(),
                    token: gateway.token.clone(),
                });
            }
            Err(_) => return Err("gateway_exited".to_string()),
        }
    }

    let addr = gateway_listen_addr();
    let token = std::env::var("PRODEX_GATEWAY_TOKEN").unwrap_or_else(|_| generate_id("gw"));
    let mut args = gateway_args(&addr);
    ensure_arg(&mut args, "--smart-context");
    if !has_arg_with_value(&args, "--listen") {
        args.push("--listen".to_string());
        args.push(addr.clone());
    }
    let prodex = std::env::var("MULTICA_PRODEX_PATH")
        .or_else(|_| std::env::var("PRODEX_BIN"))
        .unwrap_or_else(|_| "prodex".to_string());
    let mut command = Command::new(prodex);
    command
        .args(&args)
        .env("PRODEX_GATEWAY_TOKEN", &token)
        .env("PRODEX_ALLOW_UNSAFE_CHILD_ENV", "off")
        .env("NO_PROXY", "127.0.0.1,localhost")
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null());
    maybe_seed_local_gateway_key(&mut command, &args);
    let child = command.spawn().map_err(|_| "spawn_failed".to_string())?;
    *guard = Some(GatewayProcess {
        child,
        addr: addr.clone(),
        token: token.clone(),
    });
    drop(guard);

    for _ in 0..20 {
        if probe_gateway(&addr).is_ok() {
            return Ok(GatewayHandle { addr, token });
        }
        std::thread::sleep(Duration::from_millis(100));
    }
    Err("gateway_port_closed".to_string())
}

fn gateway_args(addr: &str) -> Vec<String> {
    let raw = std::env::var("MULTICA_L2_SIDECAR_ARGS").unwrap_or_default();
    let parsed = split_args(&raw);
    if parsed.first().map(String::as_str) == Some("gateway") {
        return parsed;
    }
    let base_url = std::env::var("PRODEX_GATEWAY_UPSTREAM_BASE_URL")
        .or_else(|_| std::env::var("OPENAI_BASE_URL"))
        .unwrap_or_else(|_| "http://127.0.0.1:9".to_string());
    vec![
        "gateway".to_string(),
        "--listen".to_string(),
        addr.to_string(),
        "--base-url".to_string(),
        base_url,
    ]
}

fn split_args(raw: &str) -> Vec<String> {
    let mut args = Vec::new();
    let mut current = String::new();
    let mut quote: Option<char> = None;
    for ch in raw.chars() {
        match (quote, ch) {
            (Some(q), c) if c == q => quote = None,
            (None, '\'' | '"') => quote = Some(ch),
            (None, c) if c.is_whitespace() => {
                if !current.is_empty() {
                    args.push(std::mem::take(&mut current));
                }
            }
            _ => current.push(ch),
        }
    }
    if !current.is_empty() {
        args.push(current);
    }
    args
}

fn ensure_arg(args: &mut Vec<String>, arg: &str) {
    if !args.iter().any(|existing| existing == arg) {
        args.push(arg.to_string());
    }
}

fn has_arg_with_value(args: &[String], arg: &str) -> bool {
    args.windows(2)
        .any(|pair| pair[0] == arg && !pair[1].starts_with("--"))
}

fn gateway_listen_addr() -> String {
    std::env::var("PRODEX_GATEWAY_LISTEN").unwrap_or_else(|_| "127.0.0.1:43118".to_string())
}

fn maybe_seed_local_gateway_key(command: &mut Command, args: &[String]) {
    let has_key = std::env::var("OPENAI_API_KEY").is_ok()
        || std::env::var("OPENAI_API_KEYS").is_ok()
        || has_arg_with_value(args, "--api-key");
    if has_key {
        return;
    }
    let local_base = args
        .windows(2)
        .find(|pair| pair[0] == "--base-url" || pair[0] == "--url")
        .map(|pair| pair[1].as_str())
        .unwrap_or("");
    if local_base.starts_with("http://127.0.0.1:") || local_base.starts_with("http://localhost:") {
        command.env("OPENAI_API_KEY", "sidecar-local-probe");
    }
}

fn estimate_tokens(bytes: &[u8]) -> u64 {
    let chars = String::from_utf8_lossy(bytes).chars().count() as u64;
    (chars / 4).max(1)
}

fn extract_usage_input_tokens(value: &serde_json::Value) -> Option<u64> {
    value
        .pointer("/usage/input_tokens")
        .or_else(|| value.pointer("/usage/prompt_tokens"))
        .and_then(|v| v.as_u64())
}

fn token_reduction_percent(before: u64, after: u64) -> i64 {
    if before == 0 {
        return 0;
    }
    (((before as i128 - after as i128) * 100) / before as i128) as i64
}

fn sidecar_base_url() -> String {
    std::env::var("MULTICA_L2_PUBLIC_BASE_URL").unwrap_or_else(|_| {
        let bind = sidecar_bind_addr().lock().unwrap().clone();
        format!("http://{bind}")
    })
}

fn route(req: &mut Request) -> ResponseBox {
    if !check_auth(req) {
        return error_response(401, "unauthorized");
    }

    let path = req.url().to_string();
    let method = req.method().clone();
    match (method, path.as_str()) {
        (Method::Get, "/healthz") => handle_healthz(),
        (Method::Get, "/readyz") => handle_readyz(),
        (Method::Get, p) if p.starts_with("/v1/killswitch/status") => handle_killswitch_status(p),
        (Method::Get, p) if p.starts_with("/v1/events/stream") => handle_events_stream(p),
        (Method::Post, p) if p.starts_with("/v1/runtime/proxy") => handle_runtime_proxy(req, p),
        (Method::Post, "/v1/policy/apply") => post_json(req, handle_policy_apply),
        (Method::Post, "/v1/accounts/register") => post_json(req, handle_accounts_register),
        (Method::Post, "/v1/session/start") => post_json(req, handle_session_start),
        (Method::Post, "/v1/session/stop") => post_json(req, handle_session_stop),
        (Method::Post, "/v1/killswitch/apply") => post_json(req, handle_killswitch_apply),
        _ => error_response(404, "not found"),
    }
}

fn post_json(req: &mut Request, handle: fn(&serde_json::Value) -> ResponseBox) -> ResponseBox {
    match read_json_body(req) {
        Ok(value) => match require_contract_version(&value) {
            Ok(()) => match require_allowed_tenant(&value) {
                Ok(()) => handle(&value),
                Err(resp) => resp,
            },
            Err(resp) => resp,
        },
        Err(err) => error_response(400, &err),
    }
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let bind = args
        .get(1)
        .filter(|arg| *arg != "--help" && *arg != "-h")
        .cloned()
        .unwrap_or_else(|| DEFAULT_BIND.to_string());
    if args.iter().any(|arg| arg == "--help" || arg == "-h") {
        eprintln!("Usage: prodex-sidecar [BIND_ADDR]");
        eprintln!("Default BIND_ADDR: {DEFAULT_BIND}");
        return;
    }

    let addr: SocketAddr = bind.parse().expect("valid bind address");
    *sidecar_bind_addr().lock().unwrap() = bind.clone();
    let server = Server::http(addr).expect("failed to bind sidecar");
    eprintln!("prodex-sidecar listening on {bind}");

    for mut req in server.incoming_requests() {
        let resp = route(&mut req);
        let _ = req.respond(resp);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn split_args_preserves_quoted_values() {
        let args = split_args("gateway --listen 127.0.0.1:43119 --base-url 'http://127.0.0.1:9'");
        assert_eq!(
            args,
            vec![
                "gateway",
                "--listen",
                "127.0.0.1:43119",
                "--base-url",
                "http://127.0.0.1:9"
            ]
        );
    }

    #[test]
    fn token_reduction_percent_handles_growth_and_reduction() {
        assert_eq!(token_reduction_percent(100, 60), 40);
        assert_eq!(token_reduction_percent(100, 120), -20);
        assert_eq!(token_reduction_percent(0, 120), 0);
    }

    #[test]
    fn parse_http_response_extracts_status_and_body() {
        let parsed =
            parse_http_response(b"HTTP/1.1 200 OK\r\nContent-Length: 13\r\n\r\n{\"ok\":true}\n")
                .expect("parse response");
        assert_eq!(parsed.status, 200);
        assert_eq!(String::from_utf8(parsed.body).unwrap(), "{\"ok\":true}\n");
    }

    #[test]
    fn scrub_error_category_never_returns_raw_error() {
        assert_eq!(
            scrub_error_category("postgres://user:secret@db/prodex"),
            "connect_failed"
        );
        assert_eq!(scrub_error_category("missing_config"), "missing_config");
    }
}
