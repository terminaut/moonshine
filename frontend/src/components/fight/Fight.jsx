import { useState, useEffect } from 'react'
import { useAuth } from '../../context/AuthContext'
import { percentProgressBar } from '../../utils/calculate'
import config from '../../config'
import GoldIcon from '../GoldIcon'

export default function Fight() {
  const { user } = useAuth()
  const [rounds, setRounds] = useState([])
  const [fight, setFight] = useState({})
  const [round, setRound] = useState({})
  const [player, setPlayer] = useState({})
  const [bot, setBot] = useState({})
  const [winner, setWinner] = useState(null)
  const [points, setPoints] = useState([])
  const [attackPoint, setAttackPoint] = useState('')
  const [defensePoint, setDefensePoint] = useState('')

  useEffect(() => {
    fetchFight()
  }, [])

  const fetchFight = async () => {
    try {
      const response = await fetch(`${config.apiUrl}/fight`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      })

      if (!response.ok) {
        console.error('Failed to fetch fight')
        return
      }

      const data = await response.json()
      setDataFromResponse(data)
    } catch (error) {
      console.error('Error fetching fight:', error)
    }
  }

  const setDataFromResponse = (data) => {
    setFight(data.fight || {})
    setRounds((data.fight?.rounds || []).reverse())
    setRound(data.fight?.current_round || {})
    setPlayer(data.fight?.player || {})
    setBot(data.fight?.bot || {})
    setWinner(data.fight?.winner || null)
    setPoints(data.points || [])
    if (data.points?.length > 0) {
      setAttackPoint(data.points[data.points.length - 1])
      setDefensePoint(data.points[data.points.length - 1])
    }
  }

  const attack = async () => {
    try {
      const response = await fetch(`${config.apiUrl}/fight`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify({
          defensePoint,
          attackPoint,
        }),
      })

      if (!response.ok) {
        console.error('Failed to attack')
        return
      }

      const data = await response.json()
      setDataFromResponse(data)
    } catch (error) {
      console.error('Error attacking:', error)
    }
  }

  const resolveAttackPointClass = (point) => {
    return attackPoint === point ? 'active' : ''
  }

  const resolveDefensePointClass = (point) => {
    return defensePoint === point ? 'active' : ''
  }

  const playerHpPercent = percentProgressBar(round.player_hp || 0, player.hp || 1)
  const botHpPercent = percentProgressBar(round.bot_hp || 0, bot.hp || 1)
  const playerExpPercent = percentProgressBar(player.exp || 0, player.exp_next || 1)

  return (
    <div className="container">
      {winner && (
        <div className="winner center">
          <h1 className="text-center">Winner: {winner.name}</h1>
          {fight.winner_type === 'Player' && (
            <>
              <div className="progress">
                <div
                  className="progress-bar progress-bar-danger"
                  role="progressbar"
                  style={{ width: playerHpPercent }}
                >
                  {round.player_hp} / {player.hp} HP
                </div>
              </div>

              <div className="progress">
                <div
                  className="progress-bar progress-bar-warning"
                  role="progressbar"
                  style={{ width: playerExpPercent }}
                >
                  {player.exp} / {player.exp_next} EXP
                </div>
              </div>
            </>
          )}

          <img src={winner.avatar?.url} className="image center avatar" alt="avatar" />

          {fight.dropped_gold != null && (
            <div className="center drop">
              <b>Dropped gold:</b>
              <GoldIcon width={18} height={18} />
              {fight.dropped_gold}
            </div>
          )}

          {fight.dropped_item != null && (
            <div className="center drop">
              <b>Dropped item:</b>
              <img src={fight.dropped_item.image?.url} alt="item" />
            </div>
          )}
        </div>
      )}

      {!winner && (
        <div className="fight">
          <div className="characters row">
            <div className="col-md-6">
              <div className="character player">
                <h2>
                  {player.name}[{player.level}]
                </h2>
                <div className="progress">
                  <div
                    className="progress-bar progress-bar-danger"
                    role="progressbar"
                    style={{ width: playerHpPercent }}
                  >
                    {round.player_hp} HP
                  </div>
                </div>

                <img src={player.avatar?.url} className="image avatar" alt="avatar" />

                <div className="points">
                  <h4 className="block-title">Defense: {attackPoint}</h4>
                  <ul>
                    {points.map((point) => (
                      <li key={point}>
                        <button
                          onClick={() => setAttackPoint(point)}
                          className={`btn btn-default attack ${resolveAttackPointClass(point)}`}
                        >
                          {point}
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </div>

            <div className="col-md-6">
              <div className="character bot">
                <h2>
                  {bot.name}[{bot.level}]
                </h2>
                <div className="progress">
                  <div
                    className="progress-bar progress-bar-danger"
                    role="progressbar"
                    style={{ width: botHpPercent }}
                  >
                    {round.bot_hp} HP
                  </div>
                </div>

                <img src={bot.avatar?.url} className="image avatar" alt="avatar" />

                <div className="points">
                  <h4 className="block-title">Attack: {defensePoint}</h4>
                  <ul>
                    {points.map((point) => (
                      <li key={point}>
                        <button
                          onClick={() => setDefensePoint(point)}
                          className={`btn btn-default attack ${resolveDefensePointClass(point)}`}
                        >
                          {point}
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </div>
          </div>
          <div className="row">
            <button onClick={attack} className="btn btn-danger btn-attack">
              Attack
            </button>
          </div>
        </div>
      )}

      <div className="rounds center">
        {rounds.map((round, index) => (
          round.player_damage != null && (
            <div key={index}>
              <hr />
              <div className="damage">
                <p>
                  {player.name} has dealt <span>{round.player_damage}</span> damage to {bot.name} in the{' '}
                  <b>{round.player_attack_point}</b>.
                </p>
              </div>

              <div className="damage">
                <p>
                  {bot.name} has dealt <span>{round.bot_damage}</span> damage to {player.name} in the{' '}
                  <b>{round.bot_attack_point}</b>.
                </p>
              </div>
              <hr />
            </div>
          )
        ))}
      </div>
    </div>
  )
}
















