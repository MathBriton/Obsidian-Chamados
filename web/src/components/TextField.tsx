import { useId } from 'react'

interface TextFieldProps {
  label: string
  value: string
  onChange: (value: string) => void
  type?: string
  required?: boolean
  autoComplete?: string
  placeholder?: string
}

export function TextField({
  label,
  value,
  onChange,
  type = 'text',
  required,
  autoComplete,
  placeholder,
}: TextFieldProps) {
  const id = useId()
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        id={id}
        type={type}
        value={value}
        required={required}
        autoComplete={autoComplete}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-lg border border-slate-300 px-3 py-2 text-slate-900 outline-none transition focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
      />
    </div>
  )
}
