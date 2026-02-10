import { useState } from 'react'
import { useAuth } from '../../context/AuthContext'
import config from '../../config'

export default function Stats() {
  const { user } = useAuth()
  const [showError, setShowError] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')

  if (!user) return null

  const incrementStat = async (statName) => {
    try {
      const response = await fetch(`${config.apiUrl}/player/stats/${statName}/increase`, {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      })
      
      if (!response.ok) {
        const error = await response.text()
        setErrorMessage(error)
        setShowError(true)
        return
      }
      
      const data = await response.json()
    } catch (error) {
      setErrorMessage('Failed to increment stat')
      setShowError(true)
    }
  }

  return (
    <div className="stats">
      {config.stats.map((stat) => (
        <div key={stat} className={`${stat} stat`}>
          <img src={`/assets/images/${stat}.png`} alt={stat} />
          {user[stat] || 0}
          <a onClick={() => incrementStat(stat)} style={{ cursor: 'pointer' }}>+</a>
        </div>
      ))}

      <div className="free-stats stat">
        Free stats: {user.free_stats || 0}
      </div>

      {showError && (
        <div className="alert alert-danger" role="alert" onClick={() => setShowError(false)} style={{ cursor: 'pointer' }}>
          {errorMessage}
        </div>
      )}
    </div>
  )
}
















