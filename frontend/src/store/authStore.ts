import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '../types';

interface AuthState {
  user: User | null;
  sessionId: string | null;
  setUser: (user: User, sessionId: string) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      sessionId: null,
      setUser: (user, sessionId) => set({ user, sessionId }),
      clearAuth: () => set({ user: null, sessionId: null }),
    }),
    { name: 'chatroom-auth' }
  )
);
