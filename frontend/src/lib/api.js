const API_BASE_URL = import.meta.env.VITE_API_URL || '/api'

function getAuthHeaders() {
  const token = localStorage.getItem('token')
  return {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
  }
}

async function parseResponse(response) {
  const text = await response.text()
  const trimmed = text.trim()
  if (!trimmed) return null
  try {
    return JSON.parse(trimmed)
  } catch (e) {
    return null
  }
}

export const authAPI = {
  signUp: async (username, email, password) => {
    const response = await fetch(`${API_BASE_URL}/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, email, password }),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Sign up failed')
    }
    return data
  },

  signIn: async (username, password) => {
    const response = await fetch(`${API_BASE_URL}/auth/signin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Sign in failed')
    }
    return data
  },
}

export const userAPI = {
  getCurrentUser: async () => {
    const response = await fetch(`${API_BASE_URL}/user/me`, {
      method: 'GET',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        throw new Error('Unauthorized')
      }
      throw new Error(data?.error || 'Failed to get current user')
    }
    return data
  },

  getInventory: async () => {
    const response = await fetch(`${API_BASE_URL}/users/me/inventory`, {
      method: 'GET',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        throw new Error('Unauthorized')
      }
      throw new Error(data?.error || 'Failed to fetch inventory')
    }
    return data || []
  },

  getEquippedItems: async () => {
    const response = await fetch(`${API_BASE_URL}/users/me/equipped`, {
      method: 'GET',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        throw new Error('Unauthorized')
      }
      throw new Error(data?.error || 'Failed to fetch equipped items')
    }
    return data || {}
  },

  updateProfile: async (data) => {
    const response = await fetch(`${API_BASE_URL}/user/me`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(data),
    })

    const result = await parseResponse(response)
    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        throw new Error('Unauthorized')
      }
      throw new Error(result?.error || 'Failed to update profile')
    }
    return result
  },
}

export const equipmentAPI = {
  getByCategory: async (category, artifact = false) => {
    const params = new URLSearchParams({ category })
    if (artifact) params.set('artifact', 'true')
    const response = await fetch(`${API_BASE_URL}/equipment_items?${params}`, {
      method: 'GET',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        throw new Error('Unauthorized')
      }
      throw new Error(data?.error || 'Failed to fetch equipment items')
    }
    return data || []
  },

  buy: async (itemSlug) => {
    const response = await fetch(`${API_BASE_URL}/equipment_items/${itemSlug}/buy`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to buy item')
    }
    return data
  },

  sell: async (itemSlug) => {
    const response = await fetch(`${API_BASE_URL}/equipment_items/${itemSlug}/sell`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to sell item')
    }
    return data
  },

  takeOn: async (itemSlug) => {
    const response = await fetch(`${API_BASE_URL}/equipment_items/${itemSlug}/take_on`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to equip item')
    }
    return data
  },

  takeOff: async (slotName) => {
    const response = await fetch(`${API_BASE_URL}/equipment_items/take_off/${slotName}`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to remove item')
    }
    return data
  },
}

export const potionsAPI = {
  getAll: async () => {
    const response = await fetch(`${API_BASE_URL}/potions`, {
      method: 'GET',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        throw new Error('Unauthorized')
      }
      throw new Error(data?.error || 'Failed to fetch potions')
    }
    return data || []
  },

  buy: async (potionSlug) => {
    const response = await fetch(`${API_BASE_URL}/potions/${potionSlug}/buy`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to buy potion')
    }
    return data
  },

  sell: async (potionSlug) => {
    const response = await fetch(`${API_BASE_URL}/potions/${potionSlug}/sell`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to sell potion')
    }
    return data
  },

  takeOn: async (potionSlug) => {
    const response = await fetch(`${API_BASE_URL}/potions/${potionSlug}/take_on`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to equip potion')
    }
    return data
  },

  takeOff: async (slotName) => {
    const response = await fetch(`${API_BASE_URL}/potions/take_off/${slotName}`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to remove potion')
    }
    return data
  },

  use: async (slot) => {
    const response = await fetch(`${API_BASE_URL}/potions/use/${slot}`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to use potion')
    }
    return data
  },
}

export const avatarAPI = {
  getAll: async () => {
    const response = await fetch(`${API_BASE_URL}/avatars`, {
      method: 'GET',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      if (response.status === 401) {
        localStorage.removeItem('token')
        throw new Error('Unauthorized')
      }
      throw new Error(data?.error || 'Failed to fetch avatars')
    }
    return data || []
  },
}

export const locationAPI = {
  move: async (locationSlug) => {
    const response = await fetch(`${API_BASE_URL}/locations/${locationSlug}/move`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to move to location')
    }
    return data
  },

  moveToCell: async (locationSlug, cellSlug) => {
    const response = await fetch(`${API_BASE_URL}/locations/${locationSlug}/cells/${cellSlug}/move`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to move to cell')
    }
    return data
  },
  
  getCells: async (locationSlug) => {
    const response = await fetch(`${API_BASE_URL}/locations/${locationSlug}/cells`, {
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to get location cells')
    }
    return data
  },
}

export const botAPI = {
  getBots: async (locationSlug) => {
    const response = await fetch(`${API_BASE_URL}/bots/${locationSlug}`, {
      method: 'GET',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to fetch bots')
    }
    return data?.bots || []
  },

  attack: async (botSlug) => {
    const response = await fetch(`${API_BASE_URL}/bots/${botSlug}/attack`, {
      method: 'POST',
      headers: getAuthHeaders(),
    })

    const data = await parseResponse(response)
    if (!response.ok) {
      throw new Error(data?.error || 'Failed to attack bot')
    }
    return data
  },
}
