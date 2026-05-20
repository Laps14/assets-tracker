# syntax=docker/dockerfile:1

FROM postgres:18.3-trixie

COPY ./tracked-assets.sql /docker-entrypoint-initdb.d/tracked-assets.sql
