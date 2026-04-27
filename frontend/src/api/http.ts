import axios from 'axios';
import type { ApiResponse } from '../types';

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 10_000,
  withCredentials: true, // 自动携带 Cookie（sessionId）
});

// 响应拦截：非 0 code 统一抛出，业务层 catch 即可
http.interceptors.response.use(
  (res) => {
    const body = res.data as ApiResponse;
    if (body.code !== 0) {
      return Promise.reject(new Error(body.msg || '请求失败'));
    }
    return res;
  },
  (err) => {
    const msg = err.response?.data?.msg || err.message || '网络错误';
    return Promise.reject(new Error(msg));
  }
);

export default http;
