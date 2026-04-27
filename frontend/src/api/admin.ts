import http from './http';
import type { AdminListResult, BanInput } from '../types';

export const adminApi = {
  listUsers: (params: { page?: number; page_size?: number; status?: string; keyword?: string }) =>
    http.get<{ code: number; data: AdminListResult }>('/admin/users', { params }),

  banUser: (id: number, data: BanInput) =>
    http.post(`/admin/users/${id}/ban`, data),

  unbanUser: (id: number) =>
    http.post(`/admin/users/${id}/unban`),

  deleteUser: (id: number) =>
    http.delete(`/admin/users/${id}`),

  restoreUser: (id: number) =>
    http.post(`/admin/users/${id}/restore`),

  resetPassword: (id: number, new_password: string) =>
    http.post(`/admin/users/${id}/reset-password`, { new_password }),

  kickUser: (id: number) =>
    http.post(`/admin/users/${id}/kick`),
};
