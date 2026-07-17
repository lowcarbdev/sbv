import { useState, useEffect } from 'react'
import axios from 'axios'
import { useAuth } from '../contexts/AuthContext'
import { useNavigate } from 'react-router-dom'
import ThemeToggle from './ThemeToggle'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8085/api'

function Login() {
  const [isLogin, setIsLogin] = useState(true)
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const [oidcEnabled, setOidcEnabled] = useState(false)
  const [oidcProviderName, setOidcProviderName] = useState('SSO')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login, register } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    axios.get(`${API_BASE}/config`)
      .then((response) => {
        if (response.data.registration_enabled === false) {
          setRegistrationEnabled(false)
          setIsLogin(true)
        }
        if (response.data.oidc_enabled === true) {
          setOidcEnabled(true)
          setOidcProviderName(response.data.oidc_provider_name || 'SSO')
        }
      })
      .catch(() => {
        // If the config endpoint is unavailable, leave registration visible
      })
  }, [])

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')

    // Validation
    if (!username.trim() || !password) {
      setError('Username and password are required')
      return
    }

    if (!isLogin) {
      if (username.trim().length < 3) {
        setError('Username must be at least 3 characters')
        return
      }
      if (password.length < 6) {
        setError('Password must be at least 6 characters')
        return
      }
      if (password !== confirmPassword) {
        setError('Passwords do not match')
        return
      }
    }

    setLoading(true)

    try {
      const result = isLogin
        ? await login(username.trim(), password)
        : await register(username.trim(), password)

      if (result.success) {
        navigate('/')
      } else {
        setError(result.error || 'Authentication failed')
      }
    } catch (err) {
      setError('An unexpected error occurred')
    } finally {
      setLoading(false)
    }
  }

  const toggleMode = () => {
    setIsLogin(!isLogin)
    setError('')
    setPassword('')
    setConfirmPassword('')
  }

  return (
    <div className="min-vh-100 d-flex align-items-center justify-content-center bg-body-tertiary position-relative">
      <div className="position-absolute top-0 end-0 p-3" style={{ zIndex: 10 }}>
        <ThemeToggle variant="surface" />
      </div>
      <div className="container">
        <div className="row justify-content-center">
          <div className="col-md-6 col-lg-4">
            <div className="card shadow">
              <div className="card-body p-4">
                <div className="text-center mb-4">
                  <h2 className="h4 mb-2">
                    <svg style={{width: '2rem', height: '2rem'}} className="text-primary me-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                    </svg>
                    SMS Backup Viewer
                  </h2>
                  <p className="text-muted mb-0">
                    {isLogin ? 'Sign in to your account' : 'Create a new account'}
                  </p>
                </div>

                <form onSubmit={handleSubmit}>
                  {error && (
                    <div className="alert alert-danger" role="alert">
                      {error}
                    </div>
                  )}

                  <div className="mb-3">
                    <label htmlFor="username" className="form-label">
                      Username
                    </label>
                    <input
                      type="text"
                      className="form-control"
                      id="username"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      required
                      autoComplete="username"
                      disabled={loading}
                    />
                  </div>

                  <div className="mb-3">
                    <label htmlFor="password" className="form-label">
                      Password
                    </label>
                    <input
                      type="password"
                      className="form-control"
                      id="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      required
                      autoComplete={isLogin ? 'current-password' : 'new-password'}
                      disabled={loading}
                    />
                  </div>

                  {!isLogin && (
                    <div className="mb-3">
                      <label htmlFor="confirmPassword" className="form-label">
                        Confirm Password
                      </label>
                      <input
                        type="password"
                        className="form-control"
                        id="confirmPassword"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        required
                        autoComplete="new-password"
                        disabled={loading}
                      />
                    </div>
                  )}

                  <button
                    type="submit"
                    className="btn btn-primary w-100 mb-3"
                    disabled={loading}
                  >
                    {loading ? (
                      <>
                        <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        {isLogin ? 'Signing in...' : 'Creating account...'}
                      </>
                    ) : (
                      <>{isLogin ? 'Sign In' : 'Create Account'}</>
                    )}
                  </button>

                  {oidcEnabled && (
                    <>
                      <div className="d-flex align-items-center mb-3">
                        <hr className="flex-grow-1" />
                        <span className="px-2 text-muted small">or</span>
                        <hr className="flex-grow-1" />
                      </div>
                      <a
                        href={`${API_BASE}/auth/oidc/login`}
                        className={`btn btn-outline-primary w-100 mb-3 ${loading ? 'disabled' : ''}`}
                      >
                        <svg style={{width: '1rem', height: '1rem'}} className="me-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                        </svg>
                        Sign in with {oidcProviderName}
                      </a>
                    </>
                  )}

                  {registrationEnabled && (
                    <div className="text-center">
                      <button
                        type="button"
                        className="btn btn-link text-decoration-none"
                        onClick={toggleMode}
                        disabled={loading}
                      >
                        {isLogin
                          ? "Don't have an account? Sign up"
                          : 'Already have an account? Sign in'}
                      </button>
                    </div>
                  )}
                </form>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Login
