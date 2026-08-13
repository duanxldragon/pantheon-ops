import { describe, it, expect } from 'vitest'
import { isArcoFormValidationError, isLikelyEmailAddress } from '../../../../src/core/arco/formValidation'

describe('isArcoFormValidationError', () => {
  it('returns true for object with errors key', () => {
    expect(isArcoFormValidationError({ errors: ['required'] })).toBe(true)
  })

  it('returns true for object with errors: undefined', () => {
    expect(isArcoFormValidationError({ errors: undefined })).toBe(true)
  })

  it('returns false for null', () => {
    expect(isArcoFormValidationError(null)).toBe(false)
  })

  it('returns false for string', () => {
    expect(isArcoFormValidationError('error')).toBe(false)
  })

  it('returns false for object without errors key', () => {
    expect(isArcoFormValidationError({ message: 'oops' })).toBe(false)
  })
})

describe('isLikelyEmailAddress', () => {
  it('returns true for valid email', () => {
    expect(isLikelyEmailAddress('user@example.com')).toBe(true)
  })

  it('returns true for subdomain email', () => {
    expect(isLikelyEmailAddress('user@mail.example.com')).toBe(true)
  })

  it('returns false for missing @', () => {
    expect(isLikelyEmailAddress('userexample.com')).toBe(false)
  })

  it('returns false for @ at start', () => {
    expect(isLikelyEmailAddress('@example.com')).toBe(false)
  })

  it('returns false for @ at end', () => {
    expect(isLikelyEmailAddress('user@')).toBe(false)
  })

  it('returns false for multiple @', () => {
    expect(isLikelyEmailAddress('a@b@c.com')).toBe(false)
  })

  it('returns false for domain starting with dot', () => {
    expect(isLikelyEmailAddress('user@.example.com')).toBe(false)
  })

  it('returns false for domain ending with dot', () => {
    expect(isLikelyEmailAddress('user@example.')).toBe(false)
  })

  it('returns false for single-label domain', () => {
    expect(isLikelyEmailAddress('user@localhost')).toBe(false)
  })

  it('returns false for empty string', () => {
    expect(isLikelyEmailAddress('')).toBe(false)
  })
})
