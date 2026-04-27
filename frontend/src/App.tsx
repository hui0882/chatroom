import { useAuthStore } from './store/authStore';
import AuthPage from './pages/auth/AuthPage';
import AppLayout from './components/layout/AppLayout';

export default function App() {
  const user = useAuthStore((s) => s.user);
  return user ? <AppLayout /> : <AuthPage />;
}
