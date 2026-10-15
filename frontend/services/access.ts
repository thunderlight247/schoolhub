// Access-control service layer (expert pass follow-up). Wraps the
// /admins, /accounts, and /users/:id/... endpoints the backend mounts
// behind admin.manage (master-admin) and account.block permission gates.
// Paths are BARE — lib/api.ts already prefixes /api/v1.

import { apiGet, apiPut, apiPost } from "@/lib/api";
import type {
  AccountsResponse,
  AdminAccountRow,
  AdminsResponse,
  BlockableRole,
  InviteesResponse,
} from "@/types/access";

export function listAdmins(): Promise<AdminsResponse> {
  return apiGet<AdminsResponse>("/admins");
}

/**
 * M34 "every inviter can see invitees": the accounts the signed-in user
 * provisioned themselves — admins, staff, or the parents a teacher invited.
 * Access control is the backend filter (invited_by = caller), not a permission.
 */
export function listMyInvitees(): Promise<InviteesResponse> {
  return apiGet<InviteesResponse>("/my-invitees");
}

/** Suspend a user-invited account (PUT /my-invitees/:id/status). */
export function blockInvitee(userId: string): Promise<void> {
  return apiPut<void>(`/my-invitees/${userId}/status`, { status: "SUSPENDED" });
}

/** Restore a user-invited account to ACTIVE. */
export function unblockInvitee(userId: string): Promise<void> {
  return apiPut<void>(`/my-invitees/${userId}/status`, { status: "ACTIVE" });
}

export function listAccountsByRole(role: BlockableRole): Promise<AccountsResponse> {
  return apiGet<AccountsResponse>(`/accounts?role=${role}`);
}

/** Suspend an account — kills the signed-in user's session on their next request. */
export function blockAccount(userId: string): Promise<void> {
  return apiPut<void>(`/users/${userId}/status`, { status: "SUSPENDED" });
}

/** Restore a suspended account to ACTIVE. */
export function unblockAccount(userId: string): Promise<void> {
  return apiPut<void>(`/users/${userId}/status`, { status: "ACTIVE" });
}

/** Revoke a user's ADMIN (+MASTER_ADMIN) role. Master-admin only. */
export function revokeAdminRole(userId: string): Promise<void> {
  return apiPut<void>(`/users/${userId}/admin-role`, {});
}

/**
 * Add a new ADMIN account (M34, master-admin only): the backend provisions
 * person + user + ADMIN role in one transaction and sends the Clerk
 * invitation best-effort — the recipient creates their own password through
 * the emailed link, so no password ever flows through this call.
 */
export function createAdmin(input: {
  email: string;
  first_name: string;
  last_name: string;
}): Promise<{ admin: AdminAccountRow }> {
  return apiPost<{ admin: AdminAccountRow }>("/admins", input);
}
