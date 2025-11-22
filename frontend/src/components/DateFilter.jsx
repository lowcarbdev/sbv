import DatePicker from 'react-datepicker'
import 'react-datepicker/dist/react-datepicker.css'

function DateFilter({ startDate, endDate, minDate, maxDate, onStartDateChange, onEndDateChange }) {
  const clearDates = () => {
    onStartDateChange(null)
    onEndDateChange(null)
  }

  return (
    <div className="px-2 py-1 bg-light">
      <div className="d-flex align-items-center gap-2 flex-wrap small">
        <div className="d-flex align-items-center gap-1">
          <svg style={{width: '1rem', height: '1rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <label className="fw-semibold mb-0">From:</label>
          <DatePicker
            selected={startDate}
            onChange={onStartDateChange}
            selectsStart
            startDate={startDate}
            endDate={endDate}
            minDate={minDate}
            maxDate={maxDate}
            dateFormat="MMM d, yyyy"
            className="form-control form-control-sm"
            placeholderText="Start date"
          />
        </div>

        <div className="d-flex align-items-center gap-1">
          <label className="fw-semibold mb-0">To:</label>
          <DatePicker
            selected={endDate}
            onChange={onEndDateChange}
            selectsEnd
            startDate={startDate}
            endDate={endDate}
            minDate={startDate || minDate}
            maxDate={maxDate}
            dateFormat="MMM d, yyyy"
            className="form-control form-control-sm"
            placeholderText="End date"
          />
        </div>

        {(startDate || endDate) && (
          <button
            onClick={clearDates}
            className="btn btn-sm btn-outline-primary d-flex align-items-center gap-1"
          >
            <svg style={{width: '0.875rem', height: '0.875rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
            <span className="d-none d-sm-inline">Clear</span>
          </button>
        )}
      </div>
    </div>
  )
}

export default DateFilter
