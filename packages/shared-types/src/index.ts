// OpsOwl AI — Shared TypeScript Types
// Used by apps/web and by any TypeScript tooling in the monorepo.

// ── API Envelope ─────────────────────────────────────────────────────────────

export interface APIResponse<T> {
  data?: T;
  meta?: PageMeta;
  error?: APIError;
}

export interface PageMeta {
  cursor?: string;
  has_more: boolean;
  total?: number;
}

export interface APIError {
  code: string;
  message: string;
  details?: unknown;
}

// ── Common ────────────────────────────────────────────────────────────────────

export type UUID = string;
export type ISOTimestamp = string; // ISO 8601 UTC

export interface BaseEntity {
  id: UUID;
  created_at: ISOTimestamp;
  updated_at: ISOTimestamp;
  deleted_at?: ISOTimestamp | null;
}

// ── Users & Auth ──────────────────────────────────────────────────────────────

export type UserRole = "super_admin" | "admin" | "manager" | "analyst" | "viewer";

export interface User extends BaseEntity {
  email: string;
  full_name: string;
  role: UserRole;
  is_active: boolean;
  last_login_at?: ISOTimestamp | null;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_at: ISOTimestamp;
}

// ── Cases ─────────────────────────────────────────────────────────────────────

export type CaseStatus =
  | "draft"
  | "open"
  | "in_progress"
  | "pending_review"
  | "closed"
  | "escalated";

export type CasePriority = "critical" | "high" | "medium" | "low";

export interface Case extends BaseEntity {
  case_number: string; // OWL-2026-00001
  title: string;
  description?: string | null;
  status: CaseStatus;
  priority: CasePriority;
  assigned_to?: UUID | null;
  assigned_user?: Pick<User, "id" | "full_name" | "email"> | null;
  created_by: UUID;
  creator?: Pick<User, "id" | "full_name"> | null;
  due_date?: ISOTimestamp | null;
  sla_breach_at?: ISOTimestamp | null;
  tags: string[];
}

// ── Timeline ──────────────────────────────────────────────────────────────────

export type TimelineEventSource = "system" | "user" | "ai";

export interface TimelineEvent extends BaseEntity {
  case_id: UUID;
  event_type: string;
  actor_id?: UUID | null;
  actor?: Pick<User, "id" | "full_name"> | null;
  payload: Record<string, unknown>;
  source: TimelineEventSource;
  occurred_at: ISOTimestamp;
  metadata?: Record<string, unknown> | null;
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

export type TaskStatus = "todo" | "in_progress" | "done" | "cancelled";
export type TaskPriority = "critical" | "high" | "medium" | "low";

export interface Task extends BaseEntity {
  case_id: UUID;
  title: string;
  description?: string | null;
  status: TaskStatus;
  priority: TaskPriority;
  assigned_to?: UUID | null;
  assignee?: Pick<User, "id" | "full_name"> | null;
  created_by: UUID;
  due_date?: ISOTimestamp | null;
  completed_at?: ISOTimestamp | null;
  is_overdue: boolean;
  is_ai_suggested: boolean;
}

// ── Documents ─────────────────────────────────────────────────────────────────

export type DocumentProcessingStatus =
  | "pending"
  | "processing"
  | "completed"
  | "failed";

export interface Document extends BaseEntity {
  case_id: UUID;
  uploaded_by: UUID;
  original_filename: string;
  file_type: string;
  file_size_bytes: number;
  mime_type: string;
  processing_status: DocumentProcessingStatus;
  page_count?: number | null;
  download_url?: string | null; // presigned URL, short-lived
}

// ── AI ────────────────────────────────────────────────────────────────────────

export type AIActionType =
  | "summarize"
  | "ask"
  | "generate_note"
  | "extract_actions"
  | "pattern_detect";

export type AIReviewStatus = "pending" | "approved" | "rejected" | "edited";

export interface AILog extends BaseEntity {
  case_id: UUID;
  triggered_by: UUID;
  action_type: AIActionType;
  response_text: string;
  model_used: string;
  provider: string;
  tokens_input: number;
  tokens_output: number;
  latency_ms: number;
  confidence_score?: number | null; // 0.0–1.0
  citations: AICitation[];
  human_review_status: AIReviewStatus;
  reviewed_by?: UUID | null;
  reviewed_at?: ISOTimestamp | null;
  human_edited_response?: string | null;
}

export interface AICitation {
  doc_id: UUID;
  doc_name: string;
  page?: number | null;
  excerpt: string;
  similarity_score?: number | null;
}

// ── Notifications ─────────────────────────────────────────────────────────────

export type NotificationType =
  | "case_assigned"
  | "case_status_changed"
  | "case_sla_warning"
  | "case_sla_breached"
  | "task_assigned"
  | "task_due_soon"
  | "task_overdue"
  | "mention"
  | "ai_review_required"
  | "document_processed"
  | "document_failed";

export interface Notification extends BaseEntity {
  recipient_id: UUID;
  type: NotificationType;
  title: string;
  message: string;
  link_url?: string | null;
  is_read: boolean;
  read_at?: ISOTimestamp | null;
  case_id?: UUID | null;
}

// ── Search ────────────────────────────────────────────────────────────────────

export type SearchEntityType =
  | "case"
  | "note"
  | "comment"
  | "document"
  | "task";

export interface SearchResult {
  entity_type: SearchEntityType;
  entity_id: UUID;
  case_id?: UUID | null;
  title: string;
  snippet: string; // highlighted excerpt
  relevance_score: number;
}

export interface SearchResponse {
  results: SearchResult[];
  grouped: Partial<Record<SearchEntityType, SearchResult[]>>;
  total: number;
  query: string;
}
