from datetime import datetime

from sqlalchemy import (
    Column,
    DateTime,
    ForeignKey,
    Integer,
    String,
    Text,
    UniqueConstraint,
    Boolean,
    JSON,
)
from sqlalchemy.orm import relationship

from .db import Base


class Target(Base):
    __tablename__ = "targets"

    id = Column(Integer, primary_key=True)
    handle = Column(String(128), nullable=False, index=True)
    networks = Column(JSON, nullable=False, default=list)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    last_scraped_at = Column(DateTime, nullable=True)

    events = relationship("Event", back_populates="target", cascade="all, delete-orphan")
    runs = relationship("ScrapeRun", back_populates="target", cascade="all, delete-orphan")


class Event(Base):
    __tablename__ = "events"
    __table_args__ = (
        UniqueConstraint("target_id", "network", "external_id", name="uq_event_identity"),
    )

    id = Column(Integer, primary_key=True)
    target_id = Column(Integer, ForeignKey("targets.id", ondelete="CASCADE"), nullable=False)
    network = Column(String(32), nullable=False, index=True)
    kind = Column(String(64), nullable=False, default="post")
    external_id = Column(String(512), nullable=False)
    title = Column(Text, nullable=False)
    url = Column(Text, nullable=False)
    body = Column(Text, nullable=True)
    happened_at = Column(DateTime, nullable=False, index=True)
    fetched_at = Column(DateTime, default=datetime.utcnow, nullable=False)

    target = relationship("Target", back_populates="events")


class ScrapeRun(Base):
    __tablename__ = "scrape_runs"

    id = Column(Integer, primary_key=True)
    target_id = Column(Integer, ForeignKey("targets.id", ondelete="CASCADE"), nullable=False)
    network = Column(String(32), nullable=False)
    run_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    success = Column(Boolean, default=True, nullable=False)
    message = Column(Text, nullable=True)
    new_events = Column(Integer, default=0, nullable=False)

    target = relationship("Target", back_populates="runs")
