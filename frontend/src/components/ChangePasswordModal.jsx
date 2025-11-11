import { useState } from 'react'
import { Modal, Button, Form, Alert } from 'react-bootstrap'
import { useAuth } from '../contexts/AuthContext'

function ChangePasswordModal({ onClose, onSuccess }) {
  const { changePassword } = useAuth()
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError(null)

    // Validate fields
    if (!oldPassword || !newPassword || !confirmPassword) {
      setError('All fields are required')
      return
    }

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match')
      return
    }

    if (newPassword.length < 6) {
      setError('New password must be at least 6 characters')
      return
    }

    setLoading(true)

    try {
      const result = await changePassword(oldPassword, newPassword, confirmPassword)

      if (result.success) {
        if (onSuccess) {
          onSuccess()
        }
        onClose()
      } else {
        setError(result.error || 'Failed to change password')
      }
    } catch (err) {
      setError('Failed to change password. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal show={true} onHide={onClose} centered backdrop="static" keyboard={!loading}>
      <Modal.Header closeButton={!loading}>
        <Modal.Title className="h4 fw-bold">Change Password</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {error && (
          <Alert variant="danger" dismissible onClose={() => setError(null)}>
            {error}
          </Alert>
        )}
        <Form onSubmit={handleSubmit}>
          <Form.Group className="mb-3">
            <Form.Label>Current Password</Form.Label>
            <Form.Control
              type="password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              disabled={loading}
              autoFocus
              required
            />
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>New Password</Form.Label>
            <Form.Control
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              disabled={loading}
              required
              minLength={6}
            />
            <Form.Text className="text-muted">
              Must be at least 6 characters
            </Form.Text>
          </Form.Group>
          <Form.Group className="mb-3">
            <Form.Label>Confirm New Password</Form.Label>
            <Form.Control
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              disabled={loading}
              required
              minLength={6}
            />
          </Form.Group>
        </Form>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={onClose} disabled={loading}>
          Cancel
        </Button>
        <Button
          variant="primary"
          onClick={handleSubmit}
          disabled={loading || !oldPassword || !newPassword || !confirmPassword}
        >
          {loading ? 'Changing...' : 'Change Password'}
        </Button>
      </Modal.Footer>
    </Modal>
  )
}

export default ChangePasswordModal
