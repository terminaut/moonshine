import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { authAPI } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import './Auth.css'

export default function SignIn() {
  const navigate = useNavigate()
  const { login } = useAuth()
  const [formData, setFormData] = useState({
    username: '',
    password: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const result = await authAPI.signIn(formData.username, formData.password)
      localStorage.setItem('token', result.token)
      login(result.token, result.user)
      
      const inFight = result.user?.inFight === true || result.user?.InFight === true
      if (inFight) {
        navigate('/fight', { replace: true })
      } else {
        navigate('/locations/moonshine', { replace: true })
      }
    } catch (err) {
      const errorMessage = err.message || ''
      const lowerMessage = errorMessage.toLowerCase()
      
      if (lowerMessage === 'invalid credentials') {
        setError('Неверное имя пользователя или пароль. Попробуйте еще раз.')
      } else if (lowerMessage === 'invalid input') {
        setError('Проверьте введенные данные. Имя пользователя и пароль должны содержать от 3 до 20 символов.')
      } else {
        setError('Что-то пошло не так. Попробуйте позже.')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    })
  }

  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1>Вход</h1>
        <form onSubmit={handleSubmit}>
          {error && <div className="error-message">{error}</div>}
          <div className="form-group">
            <label htmlFor="username">Имя пользователя</label>
            <input
              type="text"
              id="username"
              name="username"
              value={formData.username}
              onChange={handleChange}
              required
              minLength={3}
              maxLength={20}
            />
          </div>
          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              type="password"
              id="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              required
              minLength={3}
              maxLength={20}
            />
          </div>
          <button type="submit" disabled={loading}>
            {loading ? 'Вход...' : 'Войти'}
          </button>
        </form>
        <p className="auth-link">
          Нет аккаунта? <Link to="/signup">Зарегистрироваться</Link>
        </p>
      </div>
    </div>
  )
}
