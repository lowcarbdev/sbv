import { formatDistanceToNow } from 'date-fns'

function ConversationList({ conversations, selectedConversation, onSelectConversation, loading }) {
  const formatDate = (date) => {
    try {
      return formatDistanceToNow(new Date(date), { addSuffix: true })
    } catch {
      return ''
    }
  }

  const truncateMessage = (message, maxLength = 50) => {
    if (!message) return ''
    if (message.length <= maxLength) return message
    return message.substring(0, maxLength).trim() + '...'
  }

  const formatPhoneNumber = (number) => {
    if (!number) return 'Unknown'

    // Handle comma-separated numbers (group conversations)
    if (number.includes(',')) {
      const numbers = number.split(',').map(n => n.trim())
      return numbers.map(n => formatSinglePhoneNumber(n)).join(', ')
    }

    return formatSinglePhoneNumber(number)
  }

  const formatSinglePhoneNumber = (number) => {
    if (!number) return 'Unknown'

    // Remove all non-digit characters
    const cleaned = number.replace(/\D/g, '')

    // Handle 11-digit numbers (e.g., +1 country code)
    if (cleaned.length === 11 && cleaned.startsWith('1')) {
      return `+1 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7)}`
    }

    // Handle 10-digit numbers (US format)
    if (cleaned.length === 10) {
      return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6)}`
    }

    // Handle other formats - try to format with spaces
    if (cleaned.length > 10) {
      // International format: +XX XXX XXX XXXX
      return `+${cleaned.slice(0, cleaned.length - 10)} ${cleaned.slice(cleaned.length - 10, cleaned.length - 7)} ${cleaned.slice(cleaned.length - 7, cleaned.length - 4)} ${cleaned.slice(cleaned.length - 4)}`
    }

    // Return original if we can't format it nicely
    return number
  }

  const shouldDisplaySubject = (subject) => {
    if (!subject) return false
    // Filter out protocol buffer/RCS subjects
    if (subject.startsWith('proto:')) return false
    return true
  }

  const getDisplayName = (conv) => {
    // If we have a valid subject, use it when contact_name is empty, "(Unknown)", or looks like an 8-digit number
    if (conv.subject && shouldDisplaySubject(conv.subject)) {
      if (!conv.contact_name || conv.contact_name === '(Unknown)' || /^\d{8}$/.test(conv.contact_name)) {
        return conv.subject
      }
    }
    // If contact_name is empty, null, or "(Unknown)", use formatted phone number
    if (!conv.contact_name || conv.contact_name === '(Unknown)') {
      return formatPhoneNumber(conv.address)
    }
    return conv.contact_name
  }

  const getConversationIcon = (type) => {
    if (type === 'call') {
      return (
        <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-success" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
        </svg>
      )
    }
    return (
      <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
      </svg>
    )
  }

  if (loading) {
    return (
      <div className="d-flex align-items-center justify-content-center h-100">
        <div className="text-center">
          <div className="spinner-border text-primary mb-3" role="status" style={{width: '3rem', height: '3rem'}}>
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted fw-medium">Loading conversations...</p>
        </div>
      </div>
    )
  }

  if (conversations.length === 0) {
    return (
      <div className="d-flex align-items-center justify-content-center h-100 text-muted p-4">
        <div className="text-center">
          <svg style={{width: '4rem', height: '4rem'}} className="mx-auto mb-3 text-secondary opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
          </svg>
          <p className="fw-medium text-dark">No conversations found</p>
          <p className="small mt-2">Upload a backup file to get started</p>
        </div>
      </div>
    )
  }

  return (
    <div className="list-group list-group-flush">
      {conversations.map((conv, index) => {
        const isSelected = selectedConversation &&
          selectedConversation.address === conv.address &&
          selectedConversation.type === conv.type

        return (
          <div
            key={`${conv.type}-${conv.address}-${index}`}
            onClick={() => onSelectConversation(conv)}
            className={`list-group-item list-group-item-action ${
              isSelected ? 'active' : ''
            }`}
            style={{cursor: 'pointer'}}
          >
            <div className="d-flex align-items-start gap-2">
              <div className="flex-shrink-0 mt-1 p-2 rounded-circle bg-white shadow-sm">
                {getConversationIcon(conv.type)}
              </div>
              <div className="flex-fill min-w-0" style={{overflow: 'hidden'}}>
                <div className="d-flex justify-content-between align-items-baseline mb-1 gap-2">
                  <h6 className="fw-semibold mb-0 text-truncate" style={{flex: '1 1 auto', minWidth: 0}}>
                    {getDisplayName(conv)}
                  </h6>
                  <small className="text-nowrap flex-shrink-0" style={{fontSize: '0.75rem'}}>
                    {formatDate(conv.last_date)}
                  </small>
                </div>
                <p className="small mb-1 text-muted" style={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  fontSize: '0.85rem'
                }}>
                  {truncateMessage(conv.last_message, 50)}
                </p>
                <div className="d-flex align-items-center gap-1">
                  <span className="badge bg-secondary" style={{fontSize: '0.7rem'}}>
                    {conv.message_count} {conv.type === 'call' ? 'call' : 'message'}{conv.message_count !== 1 ? 's' : ''}
                  </span>
                </div>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}

export default ConversationList
