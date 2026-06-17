import { jwtDecode } from "jwt-decode";
import { create } from "zustand";

export interface AuthUser {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  fullName: string;
  initials: string;
  avatarLocalPath?: string;
  avatarUrl?: string;
}

interface AuthTokenClaims {
  sub?: string;
  email?: string;
  first_name?: string;
  last_name?: string;
  exp?: number;
}

interface AuthState {
  accessToken: string | null;
  user: AuthUser | null;
  login: (token: string) => void;
  updateUserProfile: (profile: Partial<AuthUser>) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  accessToken: null,
  user: null,
  login: (token) => {
    try {
      const claims = jwtDecode<AuthTokenClaims>(token);
      const expiresAt = claims.exp ? claims.exp * 1000 : undefined;

      if (expiresAt && expiresAt <= Date.now()) {
        get().logout();
        return;
      }

      const firstName = claims.first_name?.trim() ?? "";
      const lastName = claims.last_name?.trim() ?? "";
      const email = claims.email?.trim() ?? "";
      const fullName = [firstName, lastName].filter(Boolean).join(" ");

      localStorage.setItem("accessToken", token);
      set({
        accessToken: token,
        user: {
          id: claims.sub ?? "",
          email,
          firstName,
          lastName,
          fullName: fullName || email || "Taskify User",
          initials: initialsFromName(firstName, lastName, email),
        },
      });
    } catch {
      get().logout();
    }
  },
  updateUserProfile: (profile) => {
    const currentUser = get().user;
    if (!currentUser) {
      return;
    }

    const firstName = profile.firstName ?? currentUser.firstName;
    const lastName = profile.lastName ?? currentUser.lastName;
    const email = profile.email ?? currentUser.email;
    const profileFullName = [firstName, lastName].filter(Boolean).join(" ");
    const fullName =
      profile.fullName ?? (profileFullName || email || "Taskify User");

    set({
      user: {
        ...currentUser,
        ...profile,
        firstName,
        lastName,
        email,
        fullName,
        initials: profile.initials ?? initialsFromName(firstName, lastName, email),
      },
    });
  },
  logout: () => {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    set({ accessToken: null, user: null });
  },
}));

function initialsFromName(firstName: string, lastName: string, email: string) {
  const initials = `${firstName.charAt(0)}${lastName.charAt(0)}`.trim();
  if (initials) {
    return initials.toUpperCase();
  }

  return (email.charAt(0) || "T").toUpperCase();
}
