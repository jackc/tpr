def clean_database
  DB.tables.reject { |t| t == :schema_version }.each do |t|
    DB.run "truncate #{t} restart identity cascade"
  end
end
