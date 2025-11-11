import React, { useState, useEffect } from 'react'
import { parseVCard, formatAddress, formatBirthday } from '../utils/vcfParser'

/**
 * VCardPreview component for displaying vCard (contact) files
 */
function VCardPreview({ vcfText, messageId }) {
  const [contact, setContact] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    const loadVCard = () => {
      try {
        setLoading(true)
        setError(null)

        if (!vcfText) {
          throw new Error('No VCF data provided')
        }

        const parsedContact = parseVCard(vcfText)
        setContact(parsedContact)
      } catch (err) {
        console.error('Error loading vCard:', err)
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }

    if (vcfText) {
      loadVCard()
    }
  }, [vcfText])

  const handleDownload = () => {
    try {
      // Create blob from VCF text
      const blob = new Blob([vcfText], { type: 'text/vcard' })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${contact?.name || 'contact'}.vcf`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
    } catch (err) {
      console.error('Error downloading vCard:', err)
    }
  }

  if (loading) {
    return (
      <div className="card shadow-sm" style={{ maxWidth: '400px' }}>
        <div className="card-body text-center">
          <div className="spinner-border spinner-border-sm text-primary" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted small mb-0 mt-2">Loading contact...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="card shadow-sm border-danger" style={{ maxWidth: '400px' }}>
        <div className="card-body">
          <div className="text-danger small">
            <svg className="bi me-1" width="16" height="16" fill="currentColor">
              <use xlinkHref="#exclamation-triangle-fill" />
            </svg>
            Error loading contact: {error}
          </div>
        </div>
      </div>
    )
  }

  if (!contact) {
    return null
  }

  return (
    <div className="card shadow-sm" style={{ maxWidth: '400px' }}>
      <div className="card-body">
        {/* Header with photo and name */}
        <div className="d-flex align-items-center mb-3">
          {contact.photo ? (
            <img
              src={contact.photo}
              alt={contact.name}
              className="rounded-circle me-3"
              style={{ width: '64px', height: '64px', objectFit: 'cover' }}
            />
          ) : (
            <div
              className="rounded-circle bg-primary text-white d-flex align-items-center justify-content-center me-3"
              style={{ width: '64px', height: '64px', fontSize: '24px', fontWeight: 'bold' }}
            >
              {contact.name ? contact.name.charAt(0).toUpperCase() : '?'}
            </div>
          )}
          <div className="flex-grow-1">
            <h5 className="card-title mb-1">{contact.name || contact.formattedName || 'Unknown Contact'}</h5>
            {contact.title && <p className="text-muted small mb-0">{contact.title}</p>}
            {contact.organization && <p className="text-muted small mb-0">{contact.organization}</p>}
          </div>
        </div>

        {/* Contact Details */}
        <div className="vcard-details">
          {/* Phone Numbers */}
          {contact.phoneNumbers.length > 0 && (
            <div className="mb-3">
              <div className="small text-muted fw-bold mb-1">
                <svg className="bi me-1" width="14" height="14" fill="currentColor">
                  <path d="M3.654 1.328a.678.678 0 0 0-1.015-.063L1.605 2.3c-.483.484-.661 1.169-.45 1.77a17.568 17.568 0 0 0 4.168 6.608 17.569 17.569 0 0 0 6.608 4.168c.601.211 1.286.033 1.77-.45l1.034-1.034a.678.678 0 0 0-.063-1.015l-2.307-1.794a.678.678 0 0 0-.58-.122l-2.19.547a1.745 1.745 0 0 1-1.657-.459L5.482 8.062a1.745 1.745 0 0 1-.46-1.657l.548-2.19a.678.678 0 0 0-.122-.58L3.654 1.328zM1.884.511a1.745 1.745 0 0 1 2.612.163L6.29 2.98c.329.423.445.974.315 1.494l-.547 2.19a.678.678 0 0 0 .178.643l2.457 2.457a.678.678 0 0 0 .644.178l2.189-.547a1.745 1.745 0 0 1 1.494.315l2.306 1.794c.829.645.905 1.87.163 2.611l-1.034 1.034c-.74.74-1.846 1.065-2.877.702a18.634 18.634 0 0 1-7.01-4.42 18.634 18.634 0 0 1-4.42-7.009c-.362-1.03-.037-2.137.703-2.877L1.885.511z"/>
                </svg>
                Phone
              </div>
              {contact.phoneNumbers.map((phone, index) => (
                <div key={index} className="small mb-1">
                  <span className="text-muted">{phone.type}:</span>{' '}
                  <a href={`tel:${phone.number}`} className="text-decoration-none">
                    {phone.number}
                  </a>
                </div>
              ))}
            </div>
          )}

          {/* Email Addresses */}
          {contact.emails.length > 0 && (
            <div className="mb-3">
              <div className="small text-muted fw-bold mb-1">
                <svg className="bi me-1" width="14" height="14" fill="currentColor">
                  <path d="M0 4a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V4Zm2-1a1 1 0 0 0-1 1v.217l7 4.2 7-4.2V4a1 1 0 0 0-1-1H2Zm13 2.383-4.708 2.825L15 11.105V5.383Zm-.034 6.876-5.64-3.471L8 9.583l-1.326-.795-5.64 3.47A1 1 0 0 0 2 13h12a1 1 0 0 0 .966-.741ZM1 11.105l4.708-2.897L1 5.383v5.722Z"/>
                </svg>
                Email
              </div>
              {contact.emails.map((email, index) => (
                <div key={index} className="small mb-1">
                  <span className="text-muted">{email.type}:</span>{' '}
                  <a href={`mailto:${email.address}`} className="text-decoration-none">
                    {email.address}
                  </a>
                </div>
              ))}
            </div>
          )}

          {/* Addresses */}
          {contact.addresses.length > 0 && (
            <div className="mb-3">
              <div className="small text-muted fw-bold mb-1">
                <svg className="bi me-1" width="14" height="14" fill="currentColor">
                  <path d="M8.707 1.5a1 1 0 0 0-1.414 0L.646 8.146a.5.5 0 0 0 .708.708L2 8.207V13.5A1.5 1.5 0 0 0 3.5 15h9a1.5 1.5 0 0 0 1.5-1.5V8.207l.646.647a.5.5 0 0 0 .708-.708L13 5.793V2.5a.5.5 0 0 0-.5-.5h-1a.5.5 0 0 0-.5.5v1.293L8.707 1.5ZM13 7.207V13.5a.5.5 0 0 1-.5.5h-9a.5.5 0 0 1-.5-.5V7.207l5-5 5 5Z"/>
                </svg>
                Address
              </div>
              {contact.addresses.map((addr, index) => {
                const formatted = formatAddress(addr)
                return formatted ? (
                  <div key={index} className="small mb-1">
                    <span className="text-muted">{addr.type}:</span> {formatted}
                  </div>
                ) : null
              })}
            </div>
          )}

          {/* Birthday */}
          {contact.birthday && (
            <div className="mb-3">
              <div className="small text-muted fw-bold mb-1">
                <svg className="bi me-1" width="14" height="14" fill="currentColor">
                  <path d="M4 .5a.5.5 0 0 0-1 0V1H2a2 2 0 0 0-2 2v1h16V3a2 2 0 0 0-2-2h-1V.5a.5.5 0 0 0-1 0V1H4V.5zM16 14V5H0v9a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2zm-3.5-7h1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-1a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5z"/>
                </svg>
                Birthday
              </div>
              <div className="small">{formatBirthday(contact.birthday)}</div>
            </div>
          )}

          {/* URL */}
          {contact.url && (
            <div className="mb-3">
              <div className="small text-muted fw-bold mb-1">
                <svg className="bi me-1" width="14" height="14" fill="currentColor">
                  <path d="M4.715 6.542 3.343 7.914a3 3 0 1 0 4.243 4.243l1.828-1.829A3 3 0 0 0 8.586 5.5L8 6.086a1.002 1.002 0 0 0-.154.199 2 2 0 0 1 .861 3.337L6.88 11.45a2 2 0 1 1-2.83-2.83l.793-.792a4.018 4.018 0 0 1-.128-1.287z"/>
                  <path d="M6.586 4.672A3 3 0 0 0 7.414 9.5l.775-.776a2 2 0 0 1-.896-3.346L9.12 3.55a2 2 0 1 1 2.83 2.83l-.793.792c.112.42.155.855.128 1.287l1.372-1.372a3 3 0 1 0-4.243-4.243L6.586 4.672z"/>
                </svg>
                Website
              </div>
              <div className="small">
                <a href={contact.url} target="_blank" rel="noopener noreferrer" className="text-decoration-none">
                  {contact.url}
                </a>
              </div>
            </div>
          )}

          {/* Note */}
          {contact.note && (
            <div className="mb-3">
              <div className="small text-muted fw-bold mb-1">
                <svg className="bi me-1" width="14" height="14" fill="currentColor">
                  <path d="M5 0h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2 2 2 0 0 1-2 2H3a2 2 0 0 1-2-2h1a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V4a1 1 0 0 0-1-1H3a1 1 0 0 0-1 1H1a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v9a1 1 0 0 0 1-1V2a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1H3a2 2 0 0 1 2-2z"/>
                  <path d="M1 6v-.5a.5.5 0 0 1 1 0V6h.5a.5.5 0 0 1 0 1h-2a.5.5 0 0 1 0-1H1zm0 3v-.5a.5.5 0 0 1 1 0V9h.5a.5.5 0 0 1 0 1h-2a.5.5 0 0 1 0-1H1zm0 2.5v.5H.5a.5.5 0 0 0 0 1h2a.5.5 0 0 0 0-1H2v-.5a.5.5 0 0 0-1 0z"/>
                </svg>
                Note
              </div>
              <div className="small text-muted">{contact.note}</div>
            </div>
          )}
        </div>

        {/* Download Button */}
        <div className="d-grid">
          <button
            className="btn btn-sm btn-outline-primary"
            onClick={handleDownload}
          >
            <svg className="bi me-1" width="14" height="14" fill="currentColor">
              <path d="M.5 9.9a.5.5 0 0 1 .5.5v2.5a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-2.5a.5.5 0 0 1 1 0v2.5a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2v-2.5a.5.5 0 0 1 .5-.5z"/>
              <path d="M7.646 11.854a.5.5 0 0 0 .708 0l3-3a.5.5 0 0 0-.708-.708L8.5 10.293V1.5a.5.5 0 0 0-1 0v8.793L5.354 8.146a.5.5 0 1 0-.708.708l3 3z"/>
            </svg>
            Download Contact
          </button>
        </div>
      </div>
    </div>
  )
}

export default VCardPreview
