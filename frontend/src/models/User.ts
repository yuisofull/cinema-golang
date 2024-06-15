export interface User {
  id: string;
  name: string;
  gender: string;
  email: string;
  created_at: string;
  status: string;
  date_of_birth: string;
  phone: string;
  role: string;
  updated_at: string;
}
export interface Credential{
  email: string;
  password: string;
}

export interface Register{
  email: string;
  name: string;
  password: string;
}

export interface Account{
  created_at: string;
  expiry: number;
  token: string;
}

export interface Owner extends User{
  tier: string;
}