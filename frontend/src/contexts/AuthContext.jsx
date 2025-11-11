import { createContext, useContext, useState, useEffect } from 'react'
import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // Check if user is already authenticated on mount
  useEffect(() => {
    checkAuth()
  }, [])

  const checkAuth = async () => {
    try {
      const response = await axios.get(`${API_BASE}/auth/me`, {
        withCredentials: true
      })
      if (response.data.success) {
        setUser(response.data.user)
      }
    } catch (error) {
      // Not authenticated, that's okay
      setUser(null)
    } finally {
      setLoading(false)
    }
  }

  const login = async (username, password) => {
    setError(null)
    try {
      const response = await axios.post(
        `${API_BASE}/auth/login`,
        { username, password },
        { withCredentials: true }
      )

      if (response.data.success) {
        setUser(response.data.user)
        return { success: true }
      } else {
        setError(response.data.error || 'Login failed')
        return { success: false, error: response.data.error }
      }
    } catch (error) {
      const errorMsg = error.response?.data?.error || 'Login failed. Please try again.'
      setError(errorMsg)
      return { success: false, error: errorMsg }
    }
  }

  const register = async (username, password) => {
    setError(null)
    try {
      const response = await axios.post(
        `${API_BASE}/auth/register`,
        { username, password },
        { withCredentials: true }
      )

      if (response.data.success) {
        setUser(response.data.user)
        return { success: true }
      } else {
        setError(response.data.error || 'Registration failed')
        return { success: false, error: response.data.error }
      }
    } catch (error) {
      const errorMsg = error.response?.data?.error || 'Registration failed. Please try again.'
      setError(errorMsg)
      return { success: false, error: errorMsg }
    }
  }

  const logout = async () => {
    try {
      await axios.post(`${API_BASE}/auth/logout`, {}, { withCredentials: true })
    } catch (error) {
      console.error('Logout error:', error)
    } finally {
      setUser(null)
    }
  }

  const changePassword = async (oldPassword, newPassword, confirmPassword) => {
    setError(null)
    try {
      const response = await axios.post(
        `${API_BASE}/auth/change-password`,
        { old_password: oldPassword, new_password: newPassword, confirm_password: confirmPassword },
        { withCredentials: true }
      )

      if (response.data.success) {
        return { success: true }
      } else {
        setError(response.data.error || 'Password change failed')
        return { success: false, error: response.data.error }
      }
    } catch (error) {
      const errorMsg = error.response?.data?.error || 'Password change failed. Please try again.'
      setError(errorMsg)
      return { success: false, error: errorMsg }
    }
  }

  const value = {
    user,
    loading,
    error,
    login,
    register,
    logout,
    changePassword,
    isAuthenticated: !!user
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
