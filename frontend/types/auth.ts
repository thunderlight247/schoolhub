// Auth domain types (M23). Mirrors backend/internal/auth's user DTO — the
// same shape both POST /auth/login and GET /me return inside {"user": ...}.

export interface AuthUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: "ACTIVE" | "SUSPENDED" | "INACTIVE";
  /** M34: true = the auth middleware only lets change-password/logout/me
   *  through until the password is rotated (forced first-login change). */
  must_change_password: boolean;
  roles: string[];
  permissions: string[];
}

export interface AuthUserResponse {
  user: AuthUser;
}
