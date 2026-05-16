"""
L3 EventCenter — 事件总线
组件间通信，事件订阅/广播/历史
"""
import json
import threading
import time
from typing import Any, Callable, Optional
from dataclasses import dataclass, field
from datetime import datetime


@dataclass
class EventRecord:
    event: str
    payload: Any
    emitted_at: str = ''
    subscriber_count: int = 0

    def __post_init__(self):
        if not self.emitted_at:
            self.emitted_at = datetime.now().isoformat()

    def to_dict(self) -> dict:
        return {
            'event': self.event,
            'payload': self.payload,
            'emitted_at': self.emitted_at,
            'subscriber_count': self.subscriber_count
        }


class EventCenter:
    _instance = None
    _lock = threading.Lock()

    def __new__(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super().__new__(cls)
                    cls._instance._init()
        return cls._instance

    def _init(self):
        self._subs: dict[str, list[tuple[str, Callable]]] = {}  # event -> [(handler_name, fn)]
        self._history: list[EventRecord] = []
        self._max_history = 200

    def subscribe(self, event: str, handler_name: str, callback: Optional[Callable] = None) -> None:
        """Subscribe to an event with a named handler"""
        if callback is None:
            # Create a no-op callback that just prints
            def noop(payload):
                print(f'  [ec:{handler_name}] received {event}: {payload}')
            callback = noop

        with self._lock:
            if event not in self._subs:
                self._subs[event] = []
            self._subs[event].append((handler_name, callback))

    def unsubscribe(self, event: str, handler_name: str) -> None:
        """Unsubscribe a handler from an event"""
        with self._lock:
            if event in self._subs:
                self._subs[event] = [
                    (name, cb) for name, cb in self._subs[event]
                    if name != handler_name
                ]

    def emit(self, event: str, payload: Any = None) -> int:
        """Emit an event, return number of subscribers notified"""
        record = EventRecord(event=event, payload=payload)
        subscribers = []
        
        with self._lock:
            if event in self._subs:
                subscribers = list(self._subs[event])
            record.subscriber_count = len(subscribers)
            self._history.append(record)
            if len(self._history) > self._max_history:
                self._history = self._history[-self._max_history:]

        # Notify outside lock
        notified = 0
        for handler_name, callback in subscribers:
            try:
                callback(payload)
                notified += 1
            except Exception as e:
                print(f'[ec.emit error] {handler_name}: {e}')

        return notified

    def history(self, limit: int = 20) -> list:
        """Get event history"""
        with self._lock:
            return [r.to_dict() for r in self._history[-limit:]]

    def list_subs(self) -> dict:
        """List all subscriptions"""
        with self._lock:
            return {e: [name for name, _ in handlers] for e, handlers in self._subs.items()}

    def clear(self) -> None:
        """Clear all subscriptions and history"""
        with self._lock:
            self._subs.clear()
            self._history.clear()
