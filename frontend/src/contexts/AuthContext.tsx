import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import api from '../api/client';

interface AuthContextType {
  token: string | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  login: (apiKey: string) => Promise<void>;
  logout: () => void;
}

function parseJwtRole(token: string | null): boolean {
  if (!token) return false;
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.role === 'admin';
  } catch {
    return false;
  }
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));

  useEffect(() => {
    if (token) {
      localStorage.setItem('token', token);
    } else {
      localStorage.removeItem('token');
    }
  }, [token]);

  const login = async (apiKey: string) => {
    const response = await api.post('/auth/token', { api_key: apiKey });
    setToken(response.data.data.token);
  };

  const logout = () => {
    setToken(null);
  };

  return (
    <AuthContext.Provider value={{ token, isAuthenticated: !!token, isAdmin: parseJwtRole(token), login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth doit être utilisé dans un AuthProvider');
  }
  return context;
}
