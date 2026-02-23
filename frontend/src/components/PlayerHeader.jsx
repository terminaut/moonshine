import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import './PlayerHeader.css'

const LevelMatrix = {
  1: 0,
  2: 100,
  3: 200,
  4: 400,
  5: 800,
  6: 1500,
  7: 3000,
  8: 5000,
  9: 10000,
  10: 15000,
  11: 20000,
  12: 25000,
  13: 30000,
  14: 35000,
  15: 40000,
  16: 45000,
  17: 50000,
  18: 55000,
  19: 60000,
  20: 65000,
}

export default function PlayerHeader({ fullWidth = false, user: userProp = null }) {
  const { user: authUser } = useAuth()
  const user = userProp || authUser
  const currentHp = user?.currentHp ?? user?.current_hp ?? 0
  const maxHp = user?.hp || 20
  const userLevel = user?.level || 1
  const userExp = user?.exp || 0

  const nextLevelExp = LevelMatrix[userLevel + 1] || LevelMatrix[20]
  const expPercentage = nextLevelExp > 0 
    ? Math.round((userExp / nextLevelExp) * 100) 
    : 100

  return (
    <div className={`player-header-info ${fullWidth ? 'player-header-info-full' : ''}`}>
      <div className="player-name-level">
        <span className="player-name">{user?.username || 'Игрок'}</span>
        <span className="player-level">[<span className="player-level-number">{userLevel}</span>]</span>
      </div>
      <div className="player-hp-bar">
        <div 
          className="player-hp-fill" 
          style={{ 
            width: `${maxHp > 0 ? Math.round((currentHp / maxHp) * 100) : 0}%`,
            backgroundColor: '#dc3545'
          }}
        ></div>
        <span className="player-bar-label">HP</span>
        <span className="player-hp-text">
          {currentHp}/{maxHp}
        </span>
      </div>
      <div className="player-exp-bar">
        <div 
          className="player-exp-fill" 
          style={{ 
            width: `${Math.min(expPercentage, 100)}%`,
            backgroundColor: '#ffc107'
          }}
        ></div>
        <span className="player-bar-label">EXP</span>
        <span className="player-exp-text">
          {userExp}/{nextLevelExp}
        </span>
      </div>
      <Link 
        to="/profile" 
        className="profile-link-button" 
        title="Профиль персонажа"
      >
        <svg 
          width="24" 
          height="24" 
          viewBox="0 0 24 24" 
          fill="none" 
          xmlns="http://www.w3.org/2000/svg"
        >
          <path 
            d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" 
            fill="currentColor"
          />
        </svg>
      </Link>
    </div>
  )
}
