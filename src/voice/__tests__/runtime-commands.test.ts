import { describe, it, expect } from 'vitest';
import { matchRuntimeCommand } from '../runtime-commands';

describe('Voice Runtime Commands Matcher', () => {
  describe('Command Vocabulary (pt-BR + en-US)', () => {
    it('matches stop_all commands', () => {
      const cases = [
        'stop all',
        'stop everything',
        'shutdown session',
        'kill session',
        'kill all',
        'parar tudo',
        'parar sessão',
        'parar sessao',
        'encerrar tudo',
        'matar sessão',
        'parar tudo o que está rodando',
      ];
      for (const tc of cases) {
        expect(matchRuntimeCommand(tc)).toEqual({ action: 'stop_all' });
      }
    });

    it('matches deploy commands', () => {
      const cases = [
        'deploy',
        'deploy canvas',
        'materialize',
        'implantar',
        'publicar',
        'materializar',
        'executar deploy',
      ];
      for (const tc of cases) {
        expect(matchRuntimeCommand(tc)).toEqual({ action: 'deploy' });
      }
    });

    it('matches cost commands', () => {
      const cases = [
        'cost',
        'show cost',
        'check cost',
        'budget',
        'finops',
        'custo',
        'mostrar custo',
        'verificar custo',
        'gasto',
      ];
      for (const tc of cases) {
        expect(matchRuntimeCommand(tc)).toEqual({ action: 'cost' });
      }
    });

    it('matches status commands', () => {
      const cases = [
        'status',
        'show status',
        'check status',
        'terminal status',
        'verificar status',
        'mostrar status',
        'como estão as coisas',
        'como estao as coisas',
        'como está o status',
        'como esta o status',
      ];
      for (const tc of cases) {
        expect(matchRuntimeCommand(tc)).toEqual({ action: 'status' });
      }
    });

    it('matches kill commands and extracts targets', () => {
      expect(matchRuntimeCommand('kill supervisor')).toEqual({
        action: 'kill',
        target: { type: 'role', value: 'supervisor' },
      });
      expect(matchRuntimeCommand('kill the reviewer')).toEqual({
        action: 'kill',
        target: { type: 'role', value: 'reviewer' },
      });
      expect(matchRuntimeCommand('matar o revisor')).toEqual({
        action: 'kill',
        target: { type: 'role', value: 'reviewer' },
      });
      expect(matchRuntimeCommand('matar desenvolvedor')).toEqual({
        action: 'kill',
        target: { type: 'role', value: 'developer' },
      });
      expect(matchRuntimeCommand('delete terminal for review-term-id')).toEqual({
        action: 'kill',
        target: { type: 'id', value: 'review-term-id' },
      });
      expect(matchRuntimeCommand('deletar terminal bob')).toEqual({
        action: 'kill',
        target: { type: 'name', value: 'bob' },
      });
    });

    it('matches pause commands and extracts targets', () => {
      expect(matchRuntimeCommand('pause supervisor')).toEqual({
        action: 'pause',
        target: { type: 'role', value: 'supervisor' },
      });
      expect(matchRuntimeCommand('pausar o desenvolvedor')).toEqual({
        action: 'pause',
        target: { type: 'role', value: 'developer' },
      });
      expect(matchRuntimeCommand('pause terminal-123')).toEqual({
        action: 'pause',
        target: { type: 'id', value: 'terminal-123' },
      });
    });

    it('matches focus commands and extracts targets', () => {
      expect(matchRuntimeCommand('focus on supervisor')).toEqual({
        action: 'focus',
        target: { type: 'role', value: 'supervisor' },
      });
      expect(matchRuntimeCommand('focar no revisor')).toEqual({
        action: 'focus',
        target: { type: 'role', value: 'reviewer' },
      });
      expect(matchRuntimeCommand('go to reviewer agent')).toEqual({
        action: 'focus',
        target: { type: 'name', value: 'reviewer agent' },
      });
    });

    it('matches add_node commands', () => {
      expect(matchRuntimeCommand('add node developer')).toEqual({
        action: 'add_node',
        role: 'developer',
        provider: undefined,
      });
      expect(matchRuntimeCommand('adicionar nó supervisor no kiro')).toEqual({
        action: 'add_node',
        role: 'supervisor',
        provider: 'kiro_cli',
      });
      expect(matchRuntimeCommand('criar agente desenvolvedor na claude')).toEqual({
        action: 'add_node',
        role: 'developer',
        provider: 'claude_code',
      });
    });

    it('matches connect commands', () => {
      expect(matchRuntimeCommand('connect supervisor to developer')).toEqual({
        action: 'connect',
        source: { type: 'role', value: 'supervisor' },
        destination: { type: 'role', value: 'developer' },
      });
      expect(matchRuntimeCommand('conectar supervisor ao desenvolvedor')).toEqual({
        action: 'connect',
        source: { type: 'role', value: 'supervisor' },
        destination: { type: 'role', value: 'developer' },
      });
      expect(matchRuntimeCommand('handoff from developer to reviewer')).toEqual({
        action: 'connect',
        source: { type: 'role', value: 'developer' },
        destination: { type: 'role', value: 'reviewer' },
      });
    });
  });

  describe('Fallback', () => {
    it('returns null on unrecognized transcripts to allow NLU fall-through', () => {
      expect(matchRuntimeCommand('I want to build a canvas with 3 developers')).toBeNull();
      expect(matchRuntimeCommand('cria um novo canvas com supervisor no kiro')).toBeNull();
      expect(matchRuntimeCommand('')).toBeNull();
    });
  });

  describe('Latency Benchmark', () => {
    it('guarantees latency <= 100 ms on 50-character transcripts', () => {
      const transcript = 'I want to delete terminal for review-term-id please'; // 52 characters
      const start = performance.now();
      
      for (let i = 0; i < 1000; i++) {
        matchRuntimeCommand(transcript);
      }
      
      const end = performance.now();
      const averageDuration = (end - start) / 1000;
      
      expect(averageDuration).toBeLessThan(100);
    });
  });
});
