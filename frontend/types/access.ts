// Access-control domain types (expert pass follow-up). Mirrors
// backend/internal/adminaccounts DTOs — the rows GET /admins and
// GET /accounts return, plus the role vocabulary the pages key off.

export interface AdminAccountRow {
  /** users.id — the id every block/role mutation targets. */
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: "ACTIVE" | "SUSPENDED" | "INACTIVE" | string;
  /** MASTER_ADMIN flag — read-only; granted via migration, revoked with the ADMIN role. */
  is_master: boolean;
  last_login_at?: string | null;
}

export interface AdminsResponse {
  admins: AdminAccountRow[];
}

export interface RoleAccountRow {
  /** users.id — the id the block/unblock PUT targets. */
  id: string;
  /** people.id — links back to /guardians/:id or /staff/:id. */
  person_id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: "ACTIVE" | "SUSPENDED" | "INACTIVE" | string;
  last_login_at?: string | null;
}

export interface AccountsResponse {
  role: string;
  accounts: RoleAccountRow[];
}

/** Roles whose accounts the block endpoint can list and toggle. */
export const BLOCKABLE_ROLES = ["TEACHER", "STAFF", "PARENT", "STUDENT"] as const;

export type BlockableRole = (typeof BLOCKABLE_ROLES)[number];

/** One row of GET /my-invitees — an account the signed-in user minted (M34). */
export interface InviteeRow {
  /** users.id — the id the invitee-status PUT targets. */
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  /** Comma-joined role names, as the backend aggregates them. */
  roles: string;
  status: "ACTIVE" | "SUSPENDED" | "INACTIVE" | string;
}

export interface InviteesResponse {
  invitees: InviteeRow[];
}
