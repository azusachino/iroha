export type ConnectionAvailability = "supported" | "unsupported" | "disabled";
export type ConnectionCollection =
  | "unknown"
  | "partial"
  | "covered"
  | "covered_empty";
export type ConnectionOperation = "importing" | "idle" | "failed";
export type ConnectionFreshness =
  | "within_cadence"
  | "overdue"
  | "not_scheduled"
  | "unknown";

export interface ConnectionAction {
  kind: "retry_import" | "sync" | "send_bounded_payload";
  method: "POST";
  path: string;
  body?: Record<string, string>;
}

export interface ConnectionReceipt {
  id: string;
  raw_file_id: string;
  source_kind: string;
  ingestion_mode: string;
  received_at: string;
  observed_at?: string;
}

export interface ConnectionImport {
  id: string;
  status: "queued" | "parsing" | "completed" | "failed";
  parser_kind: string;
  error_message?: string;
  created_at: string;
  finished_at?: string;
}

export interface ConnectionCoverage {
  category: string;
  from: string;
  to: string;
  completeness: "unknown" | "partial" | "covered" | "covered_empty";
  recorded_at: string;
}

export interface Connection {
  id: string;
  provider: string;
  instance_key: string;
  display_name: string;
  availability: ConnectionAvailability;
  collection: ConnectionCollection;
  operation: ConnectionOperation;
  freshness: ConnectionFreshness;
  last_receipt?: ConnectionReceipt;
  last_import?: ConnectionImport;
  coverage: ConnectionCoverage[];
  next_actions: ConnectionAction[];
}

export interface ConnectionList {
  connections: Connection[];
}

export interface MatchingDecisionInput {
  provider: string;
  external_id: string;
  target_item_id: string;
  decision_kind: "attach" | "keep_separate" | "undo";
}

export interface MatchingDecision extends MatchingDecisionInput {
  id: string;
  created_at: string;
}
