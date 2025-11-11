/**
 * Parse vCard (VCF) format and extract contact information
 * Supports vCard 2.1, 3.0, and 4.0 formats
 */

/**
 * Parse a vCard file and return structured contact data
 * @param {string} vcfText - The raw vCard text content
 * @returns {Object} Parsed contact information
 */
export function parseVCard(vcfText) {
  const contact = {
    version: '',
    name: '',
    formattedName: '',
    phoneNumbers: [],
    emails: [],
    addresses: [],
    organization: '',
    title: '',
    photo: null,
    birthday: '',
    url: '',
    note: ''
  }

  // Split into lines and handle line folding (continuation lines starting with space/tab)
  const lines = unfoldLines(vcfText)

  for (const line of lines) {
    const [property, value] = parseVCardLine(line)

    if (!property) continue

    const { name, params } = parseProperty(property)

    switch (name.toUpperCase()) {
      case 'VERSION':
        contact.version = value
        break

      case 'FN':
        contact.formattedName = decodeValue(value, params)
        break

      case 'N':
        // N:LastName;FirstName;MiddleName;Prefix;Suffix
        const nameParts = value.split(';').map(p => decodeValue(p, params))
        if (!contact.name) {
          contact.name = [nameParts[3], nameParts[1], nameParts[2], nameParts[0], nameParts[4]]
            .filter(p => p)
            .join(' ')
        }
        break

      case 'TEL':
        contact.phoneNumbers.push({
          type: getTypeLabel(params, 'phone'),
          number: value
        })
        break

      case 'EMAIL':
        contact.emails.push({
          type: getTypeLabel(params, 'email'),
          address: value
        })
        break

      case 'ADR':
        // ADR:;;Street;City;State;ZIP;Country
        const adrParts = value.split(';').map(p => decodeValue(p, params))
        const address = {
          type: getTypeLabel(params, 'address'),
          street: adrParts[2],
          city: adrParts[3],
          state: adrParts[4],
          zip: adrParts[5],
          country: adrParts[6]
        }
        contact.addresses.push(address)
        break

      case 'ORG':
        contact.organization = decodeValue(value, params)
        break

      case 'TITLE':
        contact.title = decodeValue(value, params)
        break

      case 'PHOTO':
        contact.photo = parsePhoto(value, params)
        break

      case 'BDAY':
        contact.birthday = value
        break

      case 'URL':
        contact.url = value
        break

      case 'NOTE':
        contact.note = decodeValue(value, params)
        break
    }
  }

  // Use formatted name if name is empty
  if (!contact.name && contact.formattedName) {
    contact.name = contact.formattedName
  }

  return contact
}

/**
 * Unfold lines (handle line continuation in vCard format)
 */
function unfoldLines(text) {
  const lines = text.split(/\r?\n/)
  const unfolded = []
  let current = ''

  for (const line of lines) {
    // Line continuation: starts with space or tab
    if (line.startsWith(' ') || line.startsWith('\t')) {
      current += line.substring(1)
    } else {
      if (current) {
        unfolded.push(current)
      }
      current = line
    }
  }

  if (current) {
    unfolded.push(current)
  }

  return unfolded
}

/**
 * Parse a vCard line into property and value
 */
function parseVCardLine(line) {
  const colonIndex = line.indexOf(':')
  if (colonIndex === -1) return [null, null]

  const property = line.substring(0, colonIndex)
  const value = line.substring(colonIndex + 1)

  return [property, value]
}

/**
 * Parse property name and parameters
 * Example: "TEL;TYPE=CELL;PREF=1" => { name: "TEL", params: { TYPE: "CELL", PREF: "1" } }
 */
function parseProperty(property) {
  const parts = property.split(';')
  const name = parts[0]
  const params = {}

  for (let i = 1; i < parts.length; i++) {
    const param = parts[i]
    const eqIndex = param.indexOf('=')

    if (eqIndex === -1) {
      // vCard 2.1 style: TYPE without =
      params['TYPE'] = params['TYPE'] ? params['TYPE'] + ',' + param : param
    } else {
      const paramName = param.substring(0, eqIndex)
      const paramValue = param.substring(eqIndex + 1).replace(/^"(.*)"$/, '$1')
      params[paramName.toUpperCase()] = paramValue
    }
  }

  return { name, params }
}

/**
 * Get human-readable type label from parameters
 */
function getTypeLabel(params, context) {
  if (!params.TYPE) {
    return context === 'phone' ? 'Phone' : context === 'email' ? 'Email' : 'Address'
  }

  const types = params.TYPE.split(',').map(t => t.toUpperCase())

  // Common type mappings
  const typeMap = {
    'CELL': 'Mobile',
    'HOME': 'Home',
    'WORK': 'Work',
    'VOICE': 'Phone',
    'FAX': 'Fax',
    'PAGER': 'Pager',
    'MSG': 'Message',
    'PREF': 'Preferred',
    'INTERNET': 'Email'
  }

  const labels = types
    .map(t => typeMap[t] || t.charAt(0) + t.substring(1).toLowerCase())
    .filter(l => l !== 'Internet') // Remove generic Internet label

  return labels.join(', ') || (context === 'phone' ? 'Phone' : context === 'email' ? 'Email' : 'Address')
}

/**
 * Decode value based on encoding parameter
 */
function decodeValue(value, params) {
  if (!params.ENCODING) return value

  const encoding = params.ENCODING.toUpperCase()

  if (encoding === 'QUOTED-PRINTABLE') {
    return decodeQuotedPrintable(value)
  }

  return value
}

/**
 * Decode quoted-printable encoding
 */
function decodeQuotedPrintable(str) {
  return str
    .replace(/=\r?\n/g, '') // Remove soft line breaks
    .replace(/=([0-9A-F]{2})/gi, (match, hex) => String.fromCharCode(parseInt(hex, 16)))
}

/**
 * Parse photo data from vCard
 */
function parsePhoto(value, params) {
  const encoding = params.ENCODING ? params.ENCODING.toUpperCase() : ''
  const type = params.TYPE || params.MEDIATYPE || 'JPEG'

  if (encoding === 'BASE64' || encoding === 'B') {
    // Remove whitespace from base64 data
    const base64Data = value.replace(/\s/g, '')

    // Determine MIME type
    let mimeType = 'image/jpeg'
    const typeUpper = type.toUpperCase()

    if (typeUpper.includes('PNG')) {
      mimeType = 'image/png'
    } else if (typeUpper.includes('GIF')) {
      mimeType = 'image/gif'
    } else if (typeUpper.includes('BMP')) {
      mimeType = 'image/bmp'
    }

    return `data:${mimeType};base64,${base64Data}`
  }

  // URL reference
  if (value.startsWith('http://') || value.startsWith('https://')) {
    return value
  }

  return null
}

/**
 * Format address object to string
 */
export function formatAddress(address) {
  const parts = [
    address.street,
    address.city,
    address.state && address.zip ? `${address.state} ${address.zip}` : address.state || address.zip,
    address.country
  ].filter(p => p)

  return parts.join(', ')
}

/**
 * Format birthday to readable format
 */
export function formatBirthday(birthday) {
  if (!birthday) return ''

  // Handle different date formats
  // YYYYMMDD, YYYY-MM-DD, or --MMDD (no year)
  if (birthday.startsWith('--')) {
    const month = birthday.substring(2, 4)
    const day = birthday.substring(4, 6)
    return `${month}/${day}`
  }

  if (birthday.includes('-')) {
    const [year, month, day] = birthday.split('-')
    return `${month}/${day}/${year}`
  }

  if (birthday.length === 8) {
    const year = birthday.substring(0, 4)
    const month = birthday.substring(4, 6)
    const day = birthday.substring(6, 8)
    return `${month}/${day}/${year}`
  }

  return birthday
}
