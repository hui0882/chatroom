import http from './http';
import type { LoginResult, RegisterInput, User } from '../types';

export const authApi = {
  register: (data: RegisterInput) =>
    http.post<{ code: number; data: User }>('/auth/register', data),

  login: (username: string, password: string) =>
    http.post<{ code: number; data: LoginResult }>('/auth/login', { username, password }),

  logout: () => http.post('/auth/logout'),
};

export const userApi = {
  me: () => http.get<{ code: number; data: User }>('/user/me'),

  changePassword: (old_password: string, new_password: string) =>
    http.put('/user/password', { old_password, new_password }),
};
