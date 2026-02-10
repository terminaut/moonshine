import { useState, useEffect } from 'react'
import { useAuth } from '../../context/AuthContext'
import config from '../../config'

export default function OnlinePlayers({ onSetRecipient }) {
  const { user } = useAuth()
  const [players, setPlayers] = useState([])

  useEffect(() => {
    const fetchOnlinePlayers = async () => {
      try {
        const response = await fetch(`${config.apiUrl}/players/online`, {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
          },
        })

        if (!response.ok) {
          console.error('Failed to fetch online players')
          return
        }

        const data = await response.json()
        setPlayers(data || [])
      } catch (error) {
        console.error('Error fetching online players:', error)
      }
    }

    fetchOnlinePlayers()
    const interval = setInterval(fetchOnlinePlayers, 30000)
    
    return () => clearInterval(interval)
  }, [])

  const setRecipient = (player) => {
    if (user?.id === player.id) return
    onSetRecipient(player)
  }

  return (
    <div className="players-online col-md-4">
      <h3>Players online:</h3>
      {players.map((player) => (
        <div key={player.id} className="player">
          <p className="name">
            <b onClick={() => setRecipient(player)} style={{ cursor: 'pointer' }}>
              {player.name}
            </b>
            [{player.level}]
          </p>
        </div>
      ))}
    </div>
  )
}
















