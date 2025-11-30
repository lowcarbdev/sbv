import { useState, useEffect } from 'react'
import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function SettingsModal({ show, onClose, onSettingsUpdated }) {
  const [settings, setSettings] = useState({
    conversations: {
      show_calls: true
    }
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (show) {
      fetchSettings()
    }
  }, [show])

  const fetchSettings = async () => {
    try {
      setLoading(true)
      const response = await axios.get(`${API_BASE}/settings`)
      setSettings(response.data)
    } catch (err) {
      console.error('Failed to fetch settings:', err)
      setError('Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    try {
      setSaving(true)
      setError('')
      await axios.put(`${API_BASE}/settings`, settings)

      // Notify parent component that settings were updated
      if (onSettingsUpdated) {
        onSettingsUpdated(settings)
      }

      onClose()
    } catch (err) {
      console.error('Failed to save settings:', err)
      setError('Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  const handleToggleShowCalls = () => {
    setSettings({
      ...settings,
      conversations: {
        ...settings.conversations,
        show_calls: !settings.conversations.show_calls
      }
    })
  }

  if (!show) return null

  return (
    <div className="modal show d-block" tabIndex="-1" style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>
      <div className="modal-dialog modal-dialog-centered">
        <div className="modal-content">
          <div className="modal-header">
            <h5 className="modal-title">Settings</h5>
            <button type="button" className="btn-close" onClick={onClose} disabled={saving}></button>
          </div>
          <div className="modal-body">
            {loading ? (
              <div className="text-center py-4">
                <div className="spinner-border" role="status">
                  <span className="visually-hidden">Loading...</span>
                </div>
              </div>
            ) : (
              <>
                {error && (
                  <div className="alert alert-danger" role="alert">
                    {error}
                  </div>
                )}

                <h6 className="mb-3">Conversations</h6>

                <div className="form-check form-switch">
                  <input
                    className="form-check-input"
                    type="checkbox"
                    id="showCallsToggle"
                    checked={settings.conversations.show_calls}
                    onChange={handleToggleShowCalls}
                    disabled={saving}
                  />
                  <label className="form-check-label" htmlFor="showCallsToggle">
                    Show calls in conversation list
                  </label>
                  <div className="form-text">
                    When enabled, phone calls will appear in the conversation list alongside messages.
                  </div>
                </div>
              </>
            )}
          </div>
          <div className="modal-footer">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={onClose}
              disabled={saving}
            >
              Cancel
            </button>
            <button
              type="button"
              className="btn btn-primary"
              onClick={handleSave}
              disabled={loading || saving}
            >
              {saving ? (
                <>
                  <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default SettingsModal
