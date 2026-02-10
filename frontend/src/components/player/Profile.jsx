import { useAuth } from '../../context/AuthContext'
import { percentProgressBar } from '../../utils/calculate'
import config from '../../config'

export default function Profile({ player }) {
  const playerData = player || {}

  const hpPercent = percentProgressBar(playerData.current_hp || 0, playerData.hp || 1)
  const expPercent = percentProgressBar(playerData.exp || 0, playerData.exp_next || 1)

  const takeOffItem = async (item, type) => {
    try {
      const response = await fetch(`${config.apiUrl}/stuff/items/${item.id}/take_off`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify({ item_type: type }),
      })
      
      if (!response.ok) {
        console.error('Failed to take off item')
        return
      }
      
      const data = await response.json()
    } catch (error) {
      console.error('Error taking off item:', error)
    }
  }

  return (
    <div>
      <div className="player-name">
        {playerData.name}
        <strong>[{playerData.level}]</strong>
        <a href={`/players/${playerData.name}`}>
          <i className="fa fa-external-link"></i>
        </a>
      </div>

      <div className="progresses">
        <div className="progress">
          <div 
            className="progress-bar progress-bar-danger" 
            role="progressbar" 
            style={{ width: hpPercent }}
            aria-valuenow={playerData.current_hp}
            aria-valuemin="0"
            aria-valuemax={playerData.hp}
          >
            {playerData.current_hp} / {playerData.hp} HP
          </div>
        </div>

        <div className="progress">
          <div 
            className="progress-bar progress-bar-warning" 
            role="progressbar" 
            style={{ width: expPercent }}
            aria-valuenow={playerData.exp}
            aria-valuemin="0"
            aria-valuemax={playerData.exp_next}
          >
            {playerData.exp} / {playerData.exp_next} EXP
          </div>
        </div>
      </div>

      <div className="player">
        <div className="player-items left-column">
          {playerData.put_on_equipment_items?.map((item) => (
            <div key={item.id} className={`${item.category?.name?.toLowerCase()} item`}>
              <a onClick={() => takeOffItem(item, 'equipment')}>
                <img src={item.image?.url} alt="item" />
              </a>
            </div>
          ))}
        </div>

        <div className="avatar">
          <img src={playerData.avatar?.url} alt="avatar" />
        </div>

        <div className="player-items tools">
          {playerData.put_on_tool_items?.map((item, i) => (
            <div key={item.id} className={`item tool num-${i}`}>
              <a onClick={() => takeOffItem(item, 'tool')}>
                <img src={item.image?.url} alt="tool" />
              </a>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
















