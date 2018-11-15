cmake_minimum_required(VERSION 3.7)

set(CTEST_SOURCE_DIRECTORY "/source")
set(CTEST_BINARY_DIRECTORY "/binary/@CONFIG@")

if("clean" IN_LIST BUILD_STEPS)
  ctest_empty_binary_directory(${CTEST_BINARY_DIRECTORY})
endif()

site_name(CTEST_SITE)

set(CTEST_CUSTOM_MAXIMUM_NUMBER_OF_WARNINGS 1000)
set(CTEST_CMAKE_GENERATOR "Ninja")

set(CTEST_COVERAGE_COMMAND "$ENV{CTEST_COVERAGE_COMMAND}")
set(CTEST_MEMORYCHECK_COMMAND "$ENV{CTEST_MEMORYCHECK_COMMAND}")
set(CTEST_MEMORYCHECK_TYPE "$ENV{CTEST_MEMORYCHECK_TYPE}")

if("update" IN_LIST BUILD_STEPS)
  if(NOT EXISTS "${CTEST_SOURCE_DIRECTORY}/CMakeLists.txt")
    set(CTEST_CHECKOUT_COMMAND "$ENV{CTEST_CHECKOUT_COMMAND}")
  endif()
  find_program(CTEST_BZR_COMMAND bzr)
  find_program(CTEST_CVS_COMMAND cvs)
  find_program(CTEST_GIT_COMMAND git)
  find_program(CTEST_HG_COMMAND hg)
  find_program(CTEST_P4_COMMAND p4)
  find_program(CTEST_SVN_COMMAND svn)
endif()

if("start" IN_LIST BUILD_STEPS)
  ctest_start("@BUILD_MODEL@")
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(PARTS Start)
  endif()
else()
  ctest_start("@BUILD_MODEL@" APPEND)
endif()

if("update" IN_LIST BUILD_STEPS)
  set(CTEST_UPDATE_COMMAND "$ENV{CTEST_UPDATE_COMMAND}")
  ctest_update()
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(PARTS Update)
  endif()
endif()

if("configure" IN_LIST BUILD_STEPS)
  ctest_configure(OPTIONS "@CONFIGURE_OPTIONS@")
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(PARTS Configure)
  endif()
endif()

if("build" IN_LIST BUILD_STEPS)
  ctest_build()
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(PARTS Build)
  endif()
endif()

if("test" IN_LIST BUILD_STEPS)
  ctest_test(PARALLEL_LEVEL @NPROC@)
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(PARTS Test)
  endif()
endif()

if("coverage" IN_LIST BUILD_STEPS)
  ctest_coverage()
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(PARTS Coverage)
  endif()
endif()

if("gcovtar" IN_LIST BUILD_STEPS)
  include(CTestCoverageCollectGCOV)
  ctest_coverage_collect_gcov(TARBALL "${CTEST_BINARY_DIRECTORY}/gcov.tbz2")
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(
      CDASH_UPLOAD "${CTEST_BINARY_DIRECTORY}/gcov.tbz2"
      CDASH_UPLOAD_TYPE GcovTar
      )
  endif()
endif()

if("memcheck" IN_LIST BUILD_STEPS)
  ctest_memcheck(PARALLEL_LEVEL @NPROC@)
  if("submit" IN_LIST BUILD_STEPS)
    ctest_submit(PARTS MemCheck)
  endif()
endif()

if("install" IN_LIST BUILD_STEPS)
  execute_process(COMMAND cmake -DCMAKE_INSTALL_PREFIX=/prefix -P cmake_install.cmake
    WORKING_DIRECTORY "/binary/@CONFIG@"
    RESULT_VARIABLE ret
    )
  if(NOT ret EQUAL 0)
    message(FATAL_ERROR "Failed to run installation.")
  endif()
endif()

if("done" IN_LIST BUILD_STEPS)
  if("submit" IN_LIST BUILD_STEPS AND NOT CMAKE_VERSION VERSION_LESS "3.14")
    ctest_submit(PARTS Done)
  endif()
endif()
